package hub

import (
	"fmt"
	"log"
	"strconv"

	"github.com/floeit/floe/config"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

// Hub links events to the config rules
type Hub struct {
	hostID string
	config *config.Config // the config rules
	store  store.Store    // the thing to persist any state
	queue  *event.Queue   // the event q to route all events
	// runs contains list of runs ongoing or the archive
	// this is the only ongoing changing state the hub manages
	runs *RunStore
}

// NewHub creates a new hub with the given config
func NewHub(host string, c *config.Config, s store.Store, q *event.Queue) *Hub {
	h := &Hub{
		hostID: host,
		config: c,
		store:  s,
		queue:  q,
		runs:   newRunStore(s),
	}
	// hub subscribes to its own queue
	h.queue.Register(h)
	return h
}

// Notify is called whenever an event is sent to the hub. It
// makes the hub an Observer
func (h *Hub) Notify(e event.Event) {
	if e.RunRef == nil {
		// this is a an event from an external listener - thereby creating active flows
		err := h.activateFromSubs(e)
		if err != nil {
			log.Println(err)
		}
		return
	}
	// otherwise it is a flow specific event then dispatch it to any active flows
	h.dispatchActive(e)
}

func (h *Hub) dispatchActive(e event.Event) {
	// for all active flows find ones that match
	_, r := h.runs.findActiveRun(e.RunRef.ID)
	// no matching active run
	if r == nil {
		return
	}
	ns := h.config.FindNodeInFlow(r.Ref.FlowRef, e.Tag)
	for _, n := range ns {
		switch n.Class() {
		case config.NcTask:
			if n.TypeOfNode() == "end" { // special task type end the run
				h.endRun(r, e)
				return
			}
			h.executeNode(&r.Ref, n, e)
		case config.NcMerge:
			h.mergeEvent(r, n, e)
		}
	}
}

func (h *Hub) endRun(run *Run, e event.Event) {
	log.Printf("<%s> (%d) DONE  %s", run.Ref.FlowRef.ID, run.Ref.ID, e.Tag)
	h.runs.end(run)
}

func (h *Hub) mergeEvent(run *Run, node config.Node, e event.Event) {
	log.Printf("<%s> (%d) merge %s", run.Ref.FlowRef.ID, run.Ref.ID, e.Tag)

	waitsDone, opts := h.runs.updateWithMergeEvent(run, node.NodeRef().ID, e.Tag, e.Opts)
	// save the activeRun
	h.runs.Active.Save(activeKey, h.runs.store)
	// is the merge satisfied
	if (node.TypeOfNode() == "any" && waitsDone > 0) ||
		(node.TypeOfNode() == "all" && waitsDone == node.Waits()) {

		e := event.Event{
			RunRef:     &run.Ref,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "all"), // when merges fire they emit the all event
			Opts:       opts,
		}
		h.queue.Publish(e)
	}
}

func (h *Hub) executeNode(runRef *event.RunRef, node config.Node, e event.Event) {
	log.Printf("<%s> (%d) exec  %s", runRef.FlowRef.ID, runRef.ID, e.Tag)

	go func() {
		status, opts, err := node.Execute(e.Opts)
		if err != nil {
			log.Printf("<%s> (%d) error %v", runRef.FlowRef.ID, runRef.ID, err)
			return
		}
		// based on the int dispatch resultant events
		// first the all event - we dispatch in all result cases
		e := event.Event{
			RunRef:     runRef,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "all"),
			Opts:       opts,
		}
		h.queue.Publish(e)

		// dispatch good or bad event
		if node.IsStatusGood(status) {
			e.Tag = getTag(node, "good")
		} else {
			e.Tag = getTag(node, "bad")
		}
		h.queue.Publish(e)

		// dispatch the status specific event
		e.Tag = getTag(node, strconv.Itoa(status))
		h.queue.Publish(e)
	}()
}

// activateFromSubs will find any flows that match the event and activate them
func (h *Hub) activateFromSubs(e event.Event) error {
	fns := h.config.FindFlowsBySubs(e.Tag, e.Opts)

	// map the node config to events
	for f, ns := range fns {
		// add the active flow
		id, err := h.runs.addActiveFlow(f, h.hostID)
		if err != nil {
			return err
		}
		log.Printf("<%s> (%d) subs  %s", f.ID, id, e.Tag)
		// and then emit the subs node events
		for _, n := range ns {
			h.queue.Publish(event.Event{
				RunRef: &event.RunRef{
					FlowRef: n.FlowRef(),
					HostID:  h.hostID,
					ID:      id,
				},
				SourceNode: n.NodeRef(),
				Tag:        getTag(n, "good"), // all subs emit good events
				Opts:       e.Opts,
			})
		}
	}

	return nil
}

func getTag(node config.Node, subTag string) string {
	return fmt.Sprintf("%s.%s.%s", node.Class(), node.NodeRef().ID, subTag)
}
