package hub

import (
	"errors"
	"sync"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

const (
	pendingKey = "pending-list"
	activeKey  = "active-list"
	archiveKey = "archive-list"
)

// Todo is a triggered flow that is waiting for a slave
type Todo struct {
	Ref             event.RunRef
	InitiatingEvent event.Event
}

type merge struct {
	Waits map[string]bool // per node id - each wait event received
	Opts  nt.Opts         // merged opts from all listens
}

// Run is a specific invocation of a flow
type Run struct {
	sync.RWMutex
	Ref        event.RunRef
	ExecHost   string // the id of the host who's actually executing this run
	StartTime  time.Time
	EndTime    time.Time
	Ended      bool
	Status     string
	Good       bool
	MergeNodes map[string]merge
}

// updateWithMergeEvent ads the tag to the nodeID and returns current length of tags
// and a copy of the merge options
func (r *Run) updateWithMergeEvent(nodeID, tag string, opts nt.Opts) (int, nt.Opts) {
	r.Lock()
	defer r.Unlock()
	m, ok := r.MergeNodes[nodeID]
	if !ok {
		m = merge{
			Waits: map[string]bool{},
			Opts:  nt.Opts{},
		}
	}
	m.Waits[tag] = true
	m.Opts = nt.MergeOpts(m.Opts, opts)
	r.MergeNodes[nodeID] = m

	return len(m.Waits), nt.MergeOpts(m.Opts, nil) // merge copies the opts
}

func (r *Run) end(status string, good bool) {
	r.Lock()
	defer r.Unlock()
	r.EndTime = time.Now()
	r.Ended = true
	r.Status = status
	r.Good = good
}

// Pending is the thing that holds the list of flows waiting to be dispatched
type Pending struct {
	Counter int64 // The ID counter - TODO load in from the store on startup
	Todos   []*Todo
}

// Save saves the pending list
func (r Pending) Save(key string, s store.Store) error {
	return s.Save(key, r)
}

// Runs is a list of Run
type Runs []*Run

// Save saves the runs
func (r Runs) Save(key string, s store.Store) error {
	return s.Save(key, r)
}

// RunStore stores runs
type RunStore struct {
	sync.RWMutex

	// store to persist lists
	store store.Store

	// the list of flows waiting for a host
	Pending Pending

	// Active runs that we currently think are in progress
	Active Runs

	// Archive runs that are no longer active
	Archive Runs
}

func newRunStore(store store.Store) *RunStore {
	return &RunStore{
		store: store,
	}
}

// activate adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) findActiveRun(ref event.HostedIDRef) (int, *Run) {
	r.RLock()
	defer r.RUnlock()
	for i, run := range r.Active {
		if run.Ref.Run.Equals(ref) {
			return i, run
		}
	}
	return 0, nil
}

func (r *RunStore) updateWithMergeEvent(run *Run, nodeID, tag string, opts nt.Opts) (int, nt.Opts) {
	i, o := run.updateWithMergeEvent(nodeID, tag, opts)
	r.Lock()
	defer r.Unlock()
	r.Active.Save(activeKey, r.store)
	return i, o
}

func (r *RunStore) end(run *Run, status string, good bool) {
	run.end(status, good)

	i, _ := r.findActiveRun(run.Ref.Run)

	r.Lock()
	defer r.Unlock()

	// remove from active array dropping reference from underlying array
	copy(r.Active[i:], r.Active[i+1:])
	r.Active[len(r.Active)-1] = nil
	r.Active = r.Active[:len(r.Active)-1]

	r.Archive = append(r.Archive, run)
	r.Active.Save(activeKey, r.store)
	r.Archive.Save(archiveKey, r.store)
}

// addToPending adds the active configs to pending list, and returns the run id
func (r *RunStore) addToPending(flow config.FlowRef, hostID string, e event.Event) (event.HostedIDRef, error) {
	r.Lock()
	defer r.Unlock()
	r.Pending.Counter++
	run := event.HostedIDRef{
		HostID: hostID,
		ID:     r.Pending.Counter,
	}
	r.Pending.Todos = append(r.Pending.Todos, &Todo{
		Ref: event.RunRef{
			FlowRef: flow,
			Run:     run,
		},
		InitiatingEvent: e,
	})

	return run, r.Pending.Save(pendingKey, r.store)
}

// activate adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) activate(todo *Todo, hostID string) error {
	r.Lock()
	defer r.Unlock()

	// update the runref with this executing host
	todo.Ref.ExecHost = hostID

	r.Active = append(r.Active, &Run{
		Ref:        todo.Ref,
		StartTime:  time.Now(),
		MergeNodes: map[string]merge{},
	})

	return r.Active.Save(activeKey, r.store)
}

func (r *RunStore) allTodos() []*Todo {
	r.Lock()
	defer r.Unlock()
	t := make([]*Todo, len(r.Pending.Todos), len(r.Pending.Todos))
	copy(t, r.Pending.Todos)
	return t
}

func (r *RunStore) removeTodo(i int, todo *Todo) error {
	r.Lock()
	defer r.Unlock()
	if r.Pending.Todos[i] != todo {
		return errors.New("todo list mutation during dispatch")
	}
	copy(r.Pending.Todos[i:], r.Pending.Todos[i+1:])
	r.Pending.Todos[len(r.Pending.Todos)-1] = nil
	r.Pending.Todos = r.Pending.Todos[:len(r.Pending.Todos)-1]
	return nil
}
