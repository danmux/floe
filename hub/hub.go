package hub

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/floeit/floe/client"
	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/store"
)

const flowEndTag = "sys.end.all"

// Hub links events to the config rules
type Hub struct {
	sync.RWMutex

	basePath string         // the configured basePath for the hub
	hostID   string         // the id fo this host
	config   *config.Config // the config rules
	store    store.Store    // the thing to persist any state
	queue    *event.Queue   // the event q to route all events

	// tags
	tags []string // the tags that

	// hosts lists all the hosts
	hosts []*client.FloeHost

	// runs contains list of runs ongoing or the archive
	// this is the only ongoing changing state the hub manages
	runs *RunStore
}

// New creates a new hub with the given config
func New(host, tags, basePath, adminTok string, c *config.Config, s store.Store, q *event.Queue) *Hub {
	// create all tags
	l := strings.Split(tags, ",")
	tagList := []string{}
	for _, t := range l {
		t := strings.TrimSpace(t)
		tagList = append(tagList, t)
	}
	h := &Hub{
		hostID:   host,
		tags:     tagList,
		basePath: basePath,
		config:   c,
		store:    s,
		queue:    q,
		runs:     newRunStore(s),
	}
	// setup hosts
	h.setupHosts(adminTok)
	// hub subscribes to its own queue
	h.queue.Register(h)
	// start checking the pending queue
	go h.serviceLists()

	return h
}

// HostID returns the id for this host
func (h *Hub) HostID() string {
	return h.hostID
}

// Tags returns the server tags
func (h *Hub) Tags() []string {
	return h.tags
}

// AllClientRuns queries all hosts for their summaries for the given run ID
func (h *Hub) AllClientRuns(runID string) client.RunSummaries {
	s := client.RunSummaries{}
	for _, host := range h.hosts {
		summaries := host.GetRuns(runID)
		s.Append(summaries)
	}
	return s
}

// AllHosts returns all the hosts
func (h *Hub) AllHosts() map[string]client.HostConfig {
	h.Lock()
	defer h.Unlock()
	r := map[string]client.HostConfig{}
	for _, host := range h.hosts {
		c := host.GetConfig()
		r[c.HostID] = c
	}
	return r
}

// Config returns the config for this hub
func (h *Hub) Config() config.Config {
	return *h.config
}

func (h Hub) AllRuns(id string) (pending Runs, active Runs, archive Runs) {
	return h.runs.allRuns(id)
}

// Queue returns the hubs queue
func (h *Hub) Queue() *event.Queue {
	return h.queue
}

// Notify is called whenever an event is sent to the hub. It
// makes the hub an event.Observer
func (h *Hub) Notify(e event.Event) {
	// if the event has not been previously adopted in any pending todo then it is a trigger event
	if !e.RunRef.Adopted() {
		err := h.pendFlowFromTrigger(e)
		if err != nil {
			log.Error(err)
		}
		return
	}
	// otherwise it is a run specific event
	h.dispatchActive(e)
}

// ExecutePending executes a todo on this host - if this host has no conflicts.
// This could have been called directly if this is the only host, or could have
// been called via the server API as this host has been asked to accept the run.
// The boolean returned represents whether the flow was considered dealt with,
// meaning an attempt to start executing it occurred.
func (h *Hub) ExecutePending(todo Todo) (bool, error) {
	log.Debugf("<%s> - exec - attempt to execute pending type:<%s>", todo, todo.InitiatingEvent.Tag)

	flow, ok := h.config.FindFlow(todo.Ref.FlowRef, todo.InitiatingEvent.Tag, todo.InitiatingEvent.Opts)
	if !ok {
		return false, fmt.Errorf("pending flow not known %s, %s", todo.Ref.FlowRef, todo.InitiatingEvent.Tag)
	}

	// confirm no currently executing flows have a resource flag conflicts
	active := h.runs.activeFlows()
	log.Debugf("<%s> - exec - checking active conflicts with %d active runs", todo, len(active))
	for _, aRef := range active {
		fl := h.config.Flow(aRef)
		if fl == nil {
			log.Error("Strange that we have an active flow without a matching config", aRef)
			continue
		}
		if anyTags(fl.ResourceTags, flow.ResourceTags) {
			log.Debugf("<%s> - exec - found resource tag conflict on tags: %v with already active tags: %v",
				todo, flow.ResourceTags, fl.ResourceTags)
			return false, nil
		}
	}

	// setup the workspace config
	_, err := h.enforceWS(todo.Ref, flow.ReuseSpace)
	if err != nil {
		return false, err
	}

	// add the active flow
	err = h.runs.activate(todo, h.hostID)
	if err != nil {
		return false, err
	}

	log.Debugf("<%s> - exec - triggering %d nodes", todo, len(flow.Nodes))

	// and then emit the trigger event that were tripped when this flow was made pending
	// (more than one trigger at a time is going to be pretty rare)
	for _, n := range flow.Nodes {
		h.queue.Publish(event.Event{
			RunRef:     &todo.Ref,
			SourceNode: n.NodeRef(),
			Tag:        "trigger.good",            // all triggers emit the same event
			Opts:       todo.InitiatingEvent.Opts, // make sure we have the trigger event data
			Good:       true,                      // all trigger events that start a run must be good
		})
	}

	return true, nil
}

// serviceLists attempts to dispatch pending flows and times outs
// any active flows that are past their deadline
func (h *Hub) serviceLists() {
	for range time.Tick(time.Second) {
		err := h.distributePending()
		if err != nil {
			log.Error(err)
		}
	}
}

// distributePending loops through all pending todos assessing whether they can be run then distributes them.
func (h *Hub) distributePending() error {

	for _, p := range h.runs.allTodos() {
		log.Debugf("<%s> - pending - attempt dispatch", p)

		if len(h.hosts) == 0 {
			log.Debugf("<%s> - pending - no hosts configured running job locally", p)
			ok, err := h.ExecutePending(p)
			if err != nil {
				return err
			}
			if !ok {
				log.Debugf("<%s> - pending - could not run job locally yet", p)
			} else {
				log.Debugf("<%s> - pending - job started locally", p)
				h.runs.removeTodo(p)
			}
			continue
		}

		// as we have some hosts configured - attempt to schedule them
		flow, ok := h.config.FindFlow(p.Ref.FlowRef, p.InitiatingEvent.Tag, p.InitiatingEvent.Opts)
		if !ok {
			h.runs.removeTodo(p)
			return fmt.Errorf("pending not found %s, %s removed from todo", p, p.InitiatingEvent.Tag)
		}

		log.Debugf("<%s> - pending - found flow %s tags: %v", p, flow.Ref, flow.HostTags)

		// Find candidate hosts that have a superset of the tags for the pending flow
		candidates := []*client.FloeHost{}
		for _, host := range h.hosts {
			cfg := host.GetConfig()
			log.Debugf("<%s> - pending - testing host %s with host tags: %v", p, cfg.HostID, cfg.Tags)
			if cfg.TagsMatch(flow.HostTags) {
				log.Debugf("<%s> - pending - found matching host %s with host tags: %v", p, cfg.HostID, cfg.Tags)
				candidates = append(candidates, host)
			}
		}

		log.Debugf("<%s> - pending - found %d candidate hosts", p, len(candidates))

		// attempt to send it to any of the candidates
		launched := false
		for _, host := range candidates {
			if host.AttemptExecute(p.Ref, p.InitiatingEvent) {
				log.Debugf("<%s> - pending - executed on <%s>", p, host.GetConfig().HostID)
				// remove from our todo list
				h.runs.removeTodo(p)
				launched = true
			}
		}

		if !launched {
			log.Debugf("<%s> - pending - no available host yet", p)
		}

		// TODO check pending queue for any todo that is over age and send alert
	}
	return nil
}

// pendFlowFromTrigger uses the subscription fired event e to put a FoundFlow
// on the pending queue, storing the initial event for use as the run is executed.
func (h *Hub) pendFlowFromTrigger(e event.Event) error {
	// is this a generic sub event like a git hook, or an event specifically targetting a known flow
	var specificFlow *config.FlowRef
	if e.RunRef != nil {
		specificFlow = &e.RunRef.FlowRef
	}

	log.Debugf("attempt to trigger type:<%s> (specified flow: %v)", e.Tag, specificFlow)

	// find any Flows with subs matching this event
	found := h.config.FindFlowsBySubs(e.Tag, specificFlow, e.Opts)
	if len(found) == 0 {
		log.Debugf("no matching flow for type:'%s' (specified flow: %v)", e.Tag, specificFlow)
		return nil
	}
	// add each flow to the pending list
	for f := range found {
		ref, err := h.runs.addToPending(f, h.hostID, e)
		if err != nil {
			return err
		}
		log.Debugf("<%s> - from trigger type '%s' added to pending", ref, e.Tag)
	}
	return nil
}

// dispatchActive takes event e and routes it to a specific active flow as detailed in e
func (h *Hub) dispatchActive(e event.Event) {
	// the system event flow end removes the flow from the active list
	if e.Tag == flowEndTag {
		return
	}

	// for all active flows find ones that match
	_, r := h.runs.findActiveRun(e.RunRef.Run)
	if r == nil {
		// no matching active run - throw the events away
		log.Debugf("<%s> - dispatch - event '%s' received, but run not active (ignoring event)", e.RunRef, e.Tag)
		return
	}

	// find all specific nodes from the config that listen for this event
	found, flowExists := h.config.FindNodeInFlow(r.Ref.FlowRef, e.Tag)
	if !flowExists {
		log.Errorf("<%s> - dispatch - no flow for event '%s'", e.RunRef, e.Tag)
		// this is indeed a strange occurrence so this run is considered bad and incomplete
		h.endRun(r, e.SourceNode, e.Opts, "incomplete", false)
		return
	}

	// We got a matching flow but no nodes matched this event in the flow
	if len(found.Nodes) == 0 {
		if e.Good {
			// all good statuses should make it to a next node, so log the warning that this one has not
			// the run ended with a good node, but that was not explicitly routed so the run is considered incomplete
			log.Errorf("<%s> - dispatch - nothing listening to good event '%s' - prematurely end", e.RunRef, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, "incomplete", true)
		} else {
			// bad events un routed can implicitly trigger the end of a run, but the run
			// with the run marked bad
			log.Debugf("<%s> - dispatch - nothing listening to bad event '%s' (ending flow as bad)", e.RunRef, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, "complete", false)
		}
		return
	}

	// Fire all matching nodes
	for _, n := range found.Nodes {
		switch n.Class() {
		case config.NcTask:
			if n.TypeOfNode() == "end" { // special task type end the run
				h.endRun(r, n.NodeRef(), e.Opts, "complete", e.Good)
				return
			}
			// asynchronous execute
			go h.executeNode(&r.Ref, n, e, found.ReuseSpace)
		case config.NcMerge:
			h.mergeEvent(r, n, e)
		}
	}
}

// executeNode invokes a task node Execute
func (h *Hub) executeNode(runRef *event.RunRef, node config.Node, e event.Event, singleWs bool) {
	log.Debugf("<%s> - exec node - event tag: %s", runRef, e.Tag)

	// setup the workspace config
	ws, err := h.getWS(*runRef, singleWs)
	if err != nil {
		log.Debugf("<%s> - exec node - error getting workspace %v", runRef, err)
		return
	}

	// execute the node
	status, opts, err := node.Execute(*ws, e.Opts)
	if err != nil {
		log.Errorf("<%s> - exec node (%s) - execute produced error: %v", runRef, node.NodeRef(), err)
		// publish the fact an internal node error happened
		h.queue.Publish(event.Event{
			RunRef:     runRef,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "error"),
			Opts:       opts,
			Good:       false,
		})
		return
	}

	// construct event based on the Execute exit status
	ne := event.Event{
		RunRef:     runRef,
		SourceNode: node.NodeRef(),
		Opts:       opts,
	}
	// work out which tag this event has
	tagbit, good := node.Status(status)
	ne.Tag = getTag(node, tagbit)
	ne.Good = good
	// and publish it
	h.queue.Publish(ne)
}

// mergeEvent deals with events to a merge node
func (h *Hub) mergeEvent(run *Run, node config.Node, e event.Event) {
	log.Debugf("<%s> (%s) - merge %s", run.Ref.FlowRef, run.Ref.Run, e.Tag)

	waitsDone, opts := h.runs.updateWithMergeEvent(run, node.NodeRef().ID, e.Tag, e.Opts)
	// save the activeRun
	h.runs.active.Save(activeKey, h.runs.store)
	// is the merge satisfied
	if (node.TypeOfNode() == "any" && waitsDone == 1) || // only fire an any merge once
		(node.TypeOfNode() == "all" && waitsDone == node.Waits()) {

		e := event.Event{
			RunRef:     &run.Ref,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "good"), // when merges fire they emit the good event
			Good:       true,
			Opts:       opts,
		}
		h.queue.Publish(e)
	}
}

// endRun marks and saves this run as being complete
func (h *Hub) endRun(run *Run, source config.NodeRef, opts nt.Opts, status string, good bool) {
	log.Debugf("<%s> - END RUN (status:%s, good:%v)", run.Ref, status, good)
	h.runs.end(run, status, good)
	// publish specific end run event - so other observers know specifically that this flow finished
	e := event.Event{
		RunRef:     &run.Ref,
		SourceNode: source,
		Tag:        flowEndTag,
		Opts:       opts,
	}
	h.queue.Publish(e)
}

func getTag(node config.Node, subTag string) string {
	return fmt.Sprintf("%s.%s.%s", node.Class(), node.NodeRef().ID, subTag)
}

func (h *Hub) setupHosts(adminTok string) {
	h.Lock()
	defer h.Unlock()
	// TODO - consider host discovery via various mechanisms
	// e.g. etcd, dns, env vars or direct k8s api
	for _, hostAddr := range h.config.Common.Hosts {
		log.Debug("connecting to host", hostAddr)
		addr := hostAddr + h.config.Common.BaseURL
		h.hosts = append(h.hosts, client.New(addr, adminTok))
	}
}

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
