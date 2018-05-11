package hub

import (
	"strings"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

// ExecutePending executes a pending on this host if this host has no conflicts.
// This could have been called directly if this is the only host, or could have
// been called via the server API as a request for this host to accept the run.
// The boolean returned represents whether the flow was considered dealt with,
// meaning an attempt to start executing it occurred.
func (h *Hub) ExecutePending(pend Pend) (bool, error) {
	log.Debugf("<%s> - exec - attempt to execute pending from:<%s>", pend, pend.TriggeredNode)

	// use the flow definition as used when the pending run was created
	flow := pend.Flow

	// confirm no currently executing flows have a resource flag conflicts
	active := h.runs.activeFlows()
	log.Debugf("<%s> - exec - checking active conflicts with %d active runs", pend, len(active))
	for _, aRef := range active {
		fl := h.config.Flow(aRef)
		if fl == nil {
			log.Error("Strange that we have an active flow without a matching config", aRef)
			continue
		}
		if anyTags(fl.ResourceTags, flow.ResourceTags) {
			log.Debugf("<%s> - exec - found resource tag conflict on tags: %v with already active tags: %v",
				pend, flow.ResourceTags, fl.ResourceTags)
			return false, nil
		}
		if fl.ReuseSpace && flow.ReuseSpace {
			log.Debugf("<%s> - exec - reuse space is set true and flow is active", pend)
			return false, nil
		}
	}

	// setup the workspace config
	_, err := h.enforceWS(pend.Ref, flow.ReuseSpace)
	if err != nil {
		return false, err
	}

	// add the active flow
	err = h.activate(&pend, h.hostID)
	if err != nil {
		return false, err
	}

	log.Debugf("<%s> - exec - triggering from %s", pend, pend.TriggeredNode)

	// emit the trigger event that was tripped when this flow was made pending
	// This is the event that task nodes will be listening for.
	h.queue.Publish(event.Event{
		RunRef:     pend.Ref,
		SourceNode: pend.TriggeredNode,
		Tag:        tagGoodTrigger, // all triggers emit the same event
		Opts:       pend.Opts,      // make sure we have the trigger event data
		Good:       true,           // all trigger events that start a run must be good
	})

	return true, nil
}

// anyTags checks if any string in the subset is present in the set
func anyTags(set, subset []string) bool {
	for _, t := range subset {
		for _, ht := range set {
			if t == ht {
				return true
			}
		}
	}
	return false
}

// activate issues the state change system event and adds the run to the active list.
func (h *Hub) activate(pend *Pend, hostID string) error {
	h.queue.Publish(event.Event{
		RunRef: pend.Ref,
		Tag:    tagStateChange,
		Opts: nt.Opts{
			"action": "activate",
		},
		Good: true,
	})
	return h.runs.activate(pend, h.hostID)
}

// dispatchToActive takes event e that is already destined for this host
// and routes it to the specific active flow as detailed in e
func (h *Hub) dispatchToActive(e event.Event) {
	// We dont care about these system events
	if e.IsSystem() {
		return
	}

	// for all active flows find ones that match
	_, r := h.runs.findActiveRun(e.RunRef.Run)
	if r == nil {
		// no matching active run - throw the events away
		log.Debugf("<%s> - dispatch - event '%s' received, but run not active (ignoring event)", e.RunRef, e.Tag)
		return
	}

	// is it an inbound data requests
	if strings.HasPrefix(e.Tag, inboundPrefix) {
		// inbound data events sent to an active run must be targetting a specific node.
		// in which case the event SourceNode is the source that requested the data input, so
		// is therefore also the target in this case.

		// the flow is specified, as is the target node
		flow := h.config.Flow(e.RunRef.FlowRef)
		if flow == nil {
			log.Errorf("<%s> - dispatch - no flow for inbound data event flow: %s", e.RunRef, e.RunRef.FlowRef)
			return
		}
		// strip off the inbound prefix
		e.Tag = e.Tag[len(inboundPrefix)+1:]
		node := flow.Node(e.SourceNode.ID)
		if node == nil {
			log.Errorf("<%s> - dispatch - no node in flow for inbound data event flow: %s, node: %s", e.RunRef, e.RunRef.FlowRef, e.SourceNode.ID)
			return
		}

		// tell the node we have some more data
		h.setFormData(r, node, e.Opts)
		return
	}

	// find all specific nodes from the config that listen for this event
	matched := r.Flow.MatchTag(e.Tag)

	// We got a matching flow but no nodes are listening to this event in the flow.
	if len(matched) == 0 {
		if e.Good {
			// We had a good node that was not explicitly routed. These dangling good events are allowed
			// so that the other events can have a chance to finish, and hopefully hit the explicit
			// end node. If not then the run will be considered active until a (TODO) run timeout.

			// All good statuses should really make it to a next node, e.g. a merge node,
			// so log the that this one has not.
			log.Debugf("<%s> - dispatch - nothing listening to good event '%s' - prematurely end", e.RunRef, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, true)
		} else {
			// bad events un routed can implicitly trigger the end of a run,
			// with the run marked bad
			log.Debugf("<%s> - dispatch - nothing listening to bad event '%s' (ending flow as bad)", e.RunRef, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, false)
		}
		return
	}

	// Fire all matching nodes
	for _, n := range matched {
		log.Debugf("<%s> - dispatch - '%s' matched %s", e.RunRef, e.Tag, n.Ref)
		switch n.Class {
		case config.NcTask:
			switch nt.NType(n.TypeOfNode()) {
			case nt.NtEnd: // special task type end the run
				h.endRun(r, n.NodeRef(), e.Opts, e.Good)
				return
			case nt.NtData: // initial event triggering a data node (not targeted at specific node)
				h.setFormData(r, n, e.Opts)
			default:
				ws := h.prepareForExec(r.Ref, &e, r.Flow.ReuseSpace, r.Flow.Env)
				// asynchronous execute
				go h.executeNode(r, n, e, ws)
			}
		case config.NcMerge:
			h.mergeEvent(r, n, e)
		}
	}
}

func (h *Hub) prepareForExec(runRef event.RunRef, e *event.Event, singleWs bool, flowEnv []string) *nt.Workspace {
	// setup the workspace config
	ws, err := h.getWorkspace(runRef, singleWs)
	if err != nil {
		log.Debugf("<%s> - exec node - error getting workspace %v", runRef, err)
		return nil
	}

	// any event env with the flow level env
	mergeEnvOpts(e.Opts, flowEnv)

	return ws
}

// env vars from opts are added to the end of env passed in
func mergeEnvOpts(opts nt.Opts, env []string) {
	if opts == nil {
		return
	}
	if ev, ok := opts["env"]; ok {
		e, eok := ev.([]string)
		if !eok {
			return
		}
		env = append(env, e...)
	}
	opts["env"] = env
}

// exeNode defines the interface for a executable node
type exeNode interface {
	refNode
	Execute(*nt.Workspace, nt.Opts, chan string) (int, nt.Opts, error)
	Status(status int) (string, bool)
}

// executeNode invokes a task node Execute function for the active run issuing node execute update events
// and Execute exit events as appropriate
func (h *Hub) executeNode(run *Run, node exeNode, e event.Event, ws *nt.Workspace) {
	runRef := run.Ref
	nodeID := node.NodeRef().ID
	log.Debugf("<%s> - exec node - event tag: %s, node: %s", runRef, e.Tag, nodeID)

	// capture and emit all the node updates
	updates := make(chan string)
	go func() {
		for update := range updates {
			h.queue.Publish(event.Event{
				RunRef:     runRef,
				SourceNode: node.NodeRef(),
				Tag:        tagNodeUpdate,
				Opts: nt.Opts{
					"update": update,
				},
				Good: true,
			})

			// explicitly update any exec nodes with the ongoing execute
			h.runs.updateExecNode(run, nodeID, zt, zt, false, update)
		}
	}()

	// send the node start event
	h.publishIfActive(event.Event{
		RunRef:     runRef,
		SourceNode: node.NodeRef(),
		Tag:        tagNodeStart,
	})

	// set the start time for the node
	h.runs.updateExecNode(run, nodeID, time.Now(), zt, false, "")

	status, outOpts, err := node.Execute(ws, e.Opts, updates)
	close(updates)

	if err != nil {
		log.Errorf("<%s> - exec node (%s) - execute produced error: %v", runRef, node.NodeRef(), err)
		// publish the fact an internal node error happened
		h.publishIfActive(event.Event{
			RunRef:     runRef,
			SourceNode: node.NodeRef(),
			Tag:        node.GetTag("error"),
			Opts:       outOpts,
			Good:       false,
		})
		h.runs.updateExecNode(run, nodeID, zt, time.Now(), false, err.Error())
		return
	}

	// construct event based on the Execute exit status
	ne := event.Event{
		RunRef:     runRef,
		SourceNode: node.NodeRef(),
		Opts:       outOpts,
	}

	// construct the event tag
	tagbit, good := node.Status(status)
	ne.Tag = node.GetTag(tagbit)
	ne.Good = good

	h.runs.updateExecNode(run, nodeID, zt, time.Now(), good, "")

	// and publish it
	h.publishIfActive(ne)
}

// setFormData sets the opts form data on the active run on this host. If the form is incomplete it
// emits a system event which no other node should be listening for so will effectively
// pause the run, until later when any inbound data triggers the event for this data node.
// Ultimately either explicitly marking the node good or bad, and issuing the appropriate event.
func (h *Hub) setFormData(run *Run, node exeNode, opts nt.Opts) {
	// keep the filled in form values separate from the config opts
	// only use map[string]string opts
	vals := nt.Opts{}
	for k, v := range opts {
		if s, ok := v.(string); ok {
			vals[k] = s
		}
	}
	valOpts := nt.Opts{
		"values": vals,
	}

	// status 0 = good, 1 = bad, 2 = needs more data,
	status, outOpts, err := node.Execute(nil, valOpts, nil)
	if err != nil {
		log.Errorf("<%s> - set form data (%s) - execute produced error: %v", run.Ref, node.NodeRef(), err)
	}

	// add the form fields to the flow. if good or bad then we have enough data for a decision
	h.runs.updateDataNode(run, node.NodeRef().ID, outOpts, status == 2)

	ev := event.Event{
		RunRef:     run.Ref,
		SourceNode: node.NodeRef(),
		Opts:       outOpts,
	}
	switch status {
	case 0: // form data accepted and marking the node good
		ev.Tag = node.GetTag("good")
		ev.Good = true
		h.queue.Publish(ev)
	case 1:
		// form data accepted but mark the node bad
		ev.Tag = node.GetTag("bad")
		ev.Good = false
		h.queue.Publish(ev)
	case 2:
		// more data input is needed
		ev.Tag = tagWaitingData
		ev.Good = true
		h.queue.Publish(ev)
	}
}

// mergeEvent deals with events to a merge node
func (h *Hub) mergeEvent(run *Run, node mergeNode, e event.Event) {
	log.Debugf("<%s> (%s) - merge %s", run.Ref.FlowRef, run.Ref.Run, e.Tag)

	waitsDone, done, opts := h.runs.updateMergeNode(run, node.NodeRef().ID, e.Tag, node.TypeOfNode(), node.Waits(), e.Opts)

	h.queue.Publish(event.Event{
		RunRef:     run.Ref,
		SourceNode: node.NodeRef(),
		Tag:        tagNodeUpdate,
		Opts: nt.Opts{
			"waits": waitsDone,
		},
		Good: false,
	})

	// is the merge satisfied
	if done {
		e := event.Event{
			RunRef:     run.Ref,
			SourceNode: node.NodeRef(),
			Tag:        node.GetTag(config.SubTagGood), // when merges fire they emit the good event
			Good:       true,
			Opts:       opts,
		}
		h.publishIfActive(e)
	}
}

// endRun marks and saves this run as being complete
func (h *Hub) endRun(run *Run, source config.NodeRef, opts nt.Opts, good bool) {
	log.Debugf("<%s> - END RUN (good:%v)", run.Ref, good)
	didEndIt := h.runs.end(run, good)
	// if this end call was not the one that actually ended it then dont publish the end event
	if !didEndIt {
		return
	}
	// publish specific end run event - so other observers know specifically that this flow finished
	e := event.Event{
		RunRef:     run.Ref,
		SourceNode: source,
		Tag:        tagEndFlow,
		Opts:       opts,
		Good:       good,
	}
	h.queue.Publish(e)
}

// publishIfActive publishes the event if the run is still active
func (h *Hub) publishIfActive(e event.Event) {
	_, r := h.runs.findActiveRun(e.RunRef.Run)
	if r == nil {
		return
	}
	h.queue.Publish(e)
}
