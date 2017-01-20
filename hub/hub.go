package hub

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
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

// Queue returns the hubs queue
func (h *Hub) Queue() *event.Queue {
	return h.queue
}

// Notify is called whenever an event is sent to the hub. It
// makes the hub an Observer
func (h *Hub) Notify(e event.Event) {
	// if the event has no active run specified then it is potentially a triggering event
	if e.RunRef == nil || (e.RunRef != nil && e.RunRef.Inactive()) {
		err := h.pendFlowFromSubEvent(e)
		if err != nil {
			log.Error(err)
		}
		return
	}
	// otherwise it is a run specific event
	h.dispatchActive(e)
}

// ExecutePending executes a todo on this host - if this host has no conflicts
func (h *Hub) ExecutePending(todo *Todo) (bool, error) {
	log.Debugf("<%s> (%s) start %s", todo.Ref.FlowRef, todo.Ref.Run, todo.InitiatingEvent.Tag)
	flow, ok := h.config.FindFlow(todo.Ref.FlowRef, todo.InitiatingEvent.Tag, todo.InitiatingEvent.Opts)
	if !ok {
		return false, fmt.Errorf("pending not found %s, %s", todo.Ref.FlowRef, todo.InitiatingEvent.Tag)
	}

	spew.Dump(flow)

	// confirm no currently executing flows have a resource flag conflicts
	for _, aRef := range h.runs.activeFlows() {
		println(" - - - - - active", aRef.String())
		fl := h.config.Flow(aRef)
		if fl == nil {
			log.Error("active flow does not have a matching config", aRef)
			continue
		}
		fmt.Printf("%#v --- %#v\n", fl.ResourceTags, flow.ResourceTags)
		if anyTags(fl.ResourceTags, flow.ResourceTags) {
			return false, nil
		}
	}

	println("no conflic - running")

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

	// and then emit any subs node events that were tripped when this flow was made pending
	// (more than one trigger at a time is going to be pretty rare)
	for _, n := range flow.Nodes {
		h.queue.Publish(event.Event{
			RunRef:     &todo.Ref,
			SourceNode: n.NodeRef(),
			Tag:        getTag(n, "good"), // all subs emit good events
			Opts:       todo.InitiatingEvent.Opts,
		})
	}

	return true, nil
}

// serviceLists attempts to dispatch pending flows and times outs
// any active flows that are past their deadline
func (h *Hub) serviceLists() {
	for range time.Tick(time.Second) {
		err := h.dispatchPending()
		if err != nil {
			log.Error(err)
		}
	}
}

// dispatchPending loops through all pending todos assessing whether they can be run then distributes them.
func (h *Hub) dispatchPending() error {
	for _, p := range h.runs.allTodos() {
		flow, ok := h.config.FindFlow(p.Ref.FlowRef, p.InitiatingEvent.Tag, p.InitiatingEvent.Opts)
		if !ok {
			h.runs.removeTodo(p)
			return fmt.Errorf("pending not found %s, %s", p.Ref.FlowRef, p.InitiatingEvent.Tag)
		}

		// Find candidate hosts that have a superset of the tags for the pending flow
		candidates := []*client.FloeHost{}
		for _, host := range h.hosts {
			cfg := host.GetConfig()
			if cfg.TagsMatch(flow.HostTags) {
				candidates = append(candidates, host)
			}
		}
		// attempt to send it to any of the candidates
		for _, host := range candidates {
			if host.AttemptExecute(p.Ref, p.InitiatingEvent) {
				// remove from our todo list
				h.runs.removeTodo(p)
			}
		}
	}
	return nil
}

// pendFlowFromSubEvent uses the subscription fired event e to put a FoundFlow
// on the pending queue
func (h *Hub) pendFlowFromSubEvent(e event.Event) error {
	// is this a generic sub event like a git hook, or an event specifically targetting a known flow
	var specificFlow *config.FlowRef
	if e.RunRef != nil {
		specificFlow = &e.RunRef.FlowRef
	}
	// find any Flows with subs matching this event
	found := h.config.FindFlowsBySubs(e.Tag, specificFlow, e.Opts)
	if len(found) == 0 {
		log.Warning("no matching flow for", e.Tag, e.RunRef)
		return nil
	}
	// add each flow to the pending list
	for f := range found {
		id, err := h.runs.addToPending(f, h.hostID, e)
		if err != nil {
			return err
		}
		log.Debugf("<%s> (%s) subs %s", f, id, e.Tag)
	}
	return nil
}

// dispatchActive takes event e and routes it to a specific flow as detailed in e
func (h *Hub) dispatchActive(e event.Event) {
	// ignore the flow end event
	if e.Tag == flowEndTag {
		return
	}
	// for all active flows find ones that match
	_, r := h.runs.findActiveRun(e.RunRef.Run)
	// no matching active run - why do we have more events for an ended run
	if r == nil {
		log.Errorf("<%s> (%s) no run %s", e.RunRef.FlowRef, e.RunRef.Run, e.Tag)
		return
	}
	// find all specific nodes that listen for this event
	found, ok := h.config.FindNodeInFlow(r.Ref.FlowRef, e.Tag)
	// no flow matched this active event
	if !ok {
		log.Errorf("<%s> (%s) no flow for event %s", e.RunRef.FlowRef, e.RunRef.Run, e.Tag)
		return
	}
	// no nodes matched this event in the flow
	if len(found.Nodes) == 0 {
		if e.Good {
			// all good statuses should make it to a next node, so log the warning that this one has not
			log.Errorf("<%s> (%s) no node for good event %s", e.RunRef.FlowRef, e.RunRef.Run, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, "incomplete", true)
		} else {
			// bad events un routed can implicitly trigger the end of a run
			log.Debugf("<%s> (%s) no node for bad event %s (ending flow)", e.RunRef.FlowRef, e.RunRef.Run, e.Tag)
			h.endRun(r, e.SourceNode, e.Opts, "complete", false)
		}
		return
	}
	// otherwise do something for for all matching nodes
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
	log.Debugf("<%s> (%s) exec  %s", runRef.FlowRef, runRef.Run, e.Tag)
	// setup the workspace config
	ws, err := h.getWS(*runRef, singleWs)
	// execute the node
	status, opts, err := node.Execute(*ws, e.Opts)
	if err != nil {
		log.Debugf("<%s> (%d) error %v", runRef.FlowRef.ID, runRef.Run, err)
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
	log.Debugf("<%s> (%s) merge %s", run.Ref.FlowRef, run.Ref.Run, e.Tag)

	waitsDone, opts := h.runs.updateWithMergeEvent(run, node.NodeRef().ID, e.Tag, e.Opts)
	// save the activeRun
	h.runs.Active.Save(activeKey, h.runs.store)
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
	log.Debugf("<%s> (%s) DONE (%s, %v)", run.Ref.FlowRef, run.Ref.Run, status, good)
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
