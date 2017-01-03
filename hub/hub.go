package hub

import (
	"log"

	"fmt"

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
	runs RunStore
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
	var err error
	if e.RunRef == nil {
		// this is a an event from an external listener - thereby creating active flows
		err = h.activateFromSubs(e)
		return
	}
	// otherwise it is a flow specific event then dispatch it to any active flows
	println("got internal event", e.Tag)
	err = h.dispatchActive(e)
	if err != nil {
		log.Println(err)
	}
}

func (h *Hub) dispatchActive(e event.Event) error {
	// for all active flows find ones that match
	for _, r := range h.runs.Active {
		fmt.Println("finding nodes in active flow", r.Ref.FlowRef)
		ns := h.config.FindNodeInFlow(r.Ref.FlowRef, e.Tag)
		for _, n := range ns {
			fmt.Println("got node to exec", n.NodeRef().ID)
			h.executeNode(&r.Ref, n, e)
		}
	}
	return nil
}

func (h *Hub) executeNode(runRef *event.RunRef, node config.Node, e event.Event) {
	go func() {
		status, opts, err := node.Execute(e.Opts)
		if err != nil {
			log.Println("ERROR", err)
			return
		}
		// based on the int dispatch resultant events
		// first the all event - we dispatch in all result cases
		e := event.Event{
			RunRef:     runRef,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "all"), // all subs emit good events
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
		id, err := h.runs.AddActiveFlow(f)
		if err != nil {
			return err
		}
		// and then emit the subs node events
		for _, n := range ns {
			// spew.Dump(n)
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
