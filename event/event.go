package event

import (
	"sync"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
)

// RunRef uniquely identifies a particular run across the whole cluster
type RunRef struct {
	// FlowRef identifies the flow
	FlowRef config.FlowRef

	// HostID identifies the host that this run is in
	HostID string // which host in the cluster

	// ID is the run ID unique in the context of the Flow and Host
	ID int64
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
