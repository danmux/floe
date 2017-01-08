package event

import (
	"fmt"
	"sync"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/log"
)

// HostedIDRef is any ID unique to the host
type HostedIDRef struct {
	HostID string
	ID     int64
}

func (h HostedIDRef) String() string {
	return fmt.Sprintf("%s-%d", h.HostID, h.ID)
}

// Equals compares receiver with param rh
func (h HostedIDRef) Equals(rh HostedIDRef) bool {
	return h.HostID == rh.HostID && h.ID == rh.ID
}

// RunRef uniquely identifies a particular run across the whole cluster
type RunRef struct {
	// FlowRef identifies the flow
	FlowRef config.FlowRef

	// Ref identifies the host and id that this run was initiated by
	Run HostedIDRef

	// ExecHost is the host that is executing this event
	ExecHost string
}

// Observer defines the interface for observers.
type Observer interface {
	Notify(e Event)
}

// Event defines a moment in time thing occurring
type Event struct {
	// RunRef if this event is in the scope of a specific run
	// if nil then is a general event that could be routed to triggers
	RunRef *RunRef

	// SourceNode is the Ref of the node in the context of a RunRef
	SourceNode config.NodeRef

	// Tag is the label that helps route the event.
	// it will match Type for sub events, and Listen for others.
	Tag string

	// Unique and ordered event ID within a Run. An ID greater than another
	// ID must have happened after it within the context of the RunRef.
	// A flow initiating trigger will have ID 1.
	ID int64

	// Opts - some optional data in the event
	Opts nt.Opts
}

type Queue struct {
	sync.RWMutex

	idCounter int64
	// observers are any entities that care about events emitted from the queue
	observers []Observer
}

// Register registers an observer to this q
func (q *Queue) Register(o Observer) {
	q.observers = append(q.observers, o)
}

// Publish sends an event to all the observers
func (q *Queue) Publish(e Event) {
	var flowRef config.FlowRef
	var runID HostedIDRef
	if e.RunRef != nil {
		flowRef = e.RunRef.FlowRef
		runID = e.RunRef.Run
	}
	log.Debugf("<%s> (%s) event %s", flowRef, runID, e.Tag)

	// grab the next event ID
	var nextID int64
	q.Lock()
	q.idCounter++
	nextID = q.idCounter
	q.Unlock()

	e.ID = nextID

	// and notify all observers - in background goroutines
	for _, o := range q.observers {
		go o.Notify(e)
	}
}
