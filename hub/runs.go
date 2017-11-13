package hub

import (
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

func (t Todo) String() string {
	return t.Ref.String()
}

func (t Todo) Equal(u Todo) bool {
	return t.Ref.Equal(u.Ref)
}

// a merge record is kept per node id
type merge struct {
	Waits map[string]bool // each wait event received
	Opts  nt.Opts         // merged opts from all events
}

type data struct {
	Enabled bool    // Enabled is true if the enabling event has occured
	Opts    nt.Opts // opts from the data event
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
	MergeNodes map[string]merge // the states of the merge nodes by node id
	DataNodes  map[string]data  // the sates of any data nodes
}

// updateWithMergeEvent adds the tag to the nodeID and returns current length of tags
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

// Pending is the thing that holds the list of flows waiting to be dispatched.
// Being added to the Pending list assigned the RunRef
type pending struct {
	Counter int64 // The ID counter - TODO load in from the store on startup
	Todos   []*Todo
}

// Save saves the pending list
func (r pending) Save(key string, s store.Store) error {
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
	pending pending

	// active runs that we currently think are in progress
	active Runs

	// archive runs that are no longer active
	archive Runs
}

func newRunStore(store store.Store) *RunStore {
	return &RunStore{
		store: store,
	}
}

// findActiveRun returns the run from the active list that matches the given ref
func (r *RunStore) findActiveRun(ref event.HostedIDRef) (int, *Run) {
	r.RLock()
	defer r.RUnlock()
	for i, run := range r.active {
		if run.Ref.Run.Equals(ref) {
			return i, run
		}
	}
	return -1, nil
}

func (r *RunStore) updateWithMergeEvent(run *Run, nodeID, tag string, opts nt.Opts) (int, nt.Opts) {
	i, o := run.updateWithMergeEvent(nodeID, tag, opts)
	r.Lock()
	defer r.Unlock()
	r.active.Save(activeKey, r.store)
	return i, o
}

// end moves the run from active to archive. As a run may have many events that would end it
// only the first one does the others are ignored. Only the ending run returns true.
func (r *RunStore) end(run *Run, status string, good bool) bool{
	run.end(status, good)

	i, run := r.findActiveRun(run.Ref.Run)
	if run == nil {
		return false
	}

	r.Lock()
	defer r.Unlock()

	// remove from active array dropping reference from underlying array
	copy(r.active[i:], r.active[i+1:])
	r.active[len(r.active)-1] = nil
	r.active = r.active[:len(r.active)-1]
	r.archive = append(r.archive, run)

	r.active.Save(activeKey, r.store)
	r.archive.Save(archiveKey, r.store)

	return true
}

// addToPending adds the active configs to pending list, and returns the run id
func (r *RunStore) addToPending(flow config.FlowRef, hostID string, e event.Event) (event.RunRef, error) {
	r.Lock()
	defer r.Unlock()
	r.pending.Counter++
	run := event.HostedIDRef{
		HostID: hostID,
		ID:     r.pending.Counter,
	}
	t := &Todo{
		Ref: event.RunRef{
			FlowRef: flow,
			Run:     run,
		},
		InitiatingEvent: e,
	}
	r.pending.Todos = append(r.pending.Todos, t)

	return t.Ref, r.pending.Save(pendingKey, r.store)
}

// activeFlows returns all the flowrefs that match those currently executing
func (r *RunStore) activeFlows() []config.FlowRef {
	r.RLock()
	defer r.RUnlock()
	res := []config.FlowRef{}
	for _, run := range r.active {
		res = append(res, run.Ref.FlowRef)
	}
	return res
}

// activate adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) activate(todo Todo, hostID string) error {
	r.Lock()
	defer r.Unlock()

	// update the runref with this executing host
	todo.Ref.ExecHost = hostID

	r.active = append(r.active, &Run{
		Ref:        todo.Ref,
		StartTime:  time.Now(),
		MergeNodes: map[string]merge{},
	})

	return r.active.Save(activeKey, r.store)
}

func (r *RunStore) allTodos() []Todo {
	r.Lock()
	defer r.Unlock()
	t := make([]Todo, len(r.pending.Todos))
	for i, todo := range r.pending.Todos {
		t[i] = *todo
	}
	return t
}

// removeTodo returns true if the given todo is removed from the pending list
func (r *RunStore) removeTodo(todo Todo) (bool, error) {
	r.Lock()
	defer r.Unlock()

	for i, td := range r.pending.Todos {
		if td.Equal(todo) {
			// slide them down
			copy(r.pending.Todos[i:], r.pending.Todos[i+1:])
			// explicitly drop the reference to the one left at the end
			r.pending.Todos[len(r.pending.Todos)-1] = nil
			// and remove it from the slice
			r.pending.Todos = r.pending.Todos[:len(r.pending.Todos)-1]

			// save the whole pending list
			return true, r.pending.Save(pendingKey, r.store)
		}
	}
	// If the todo is not found then there is nothing to worry about
	// it is already removed
	return false, nil
}

func (r *RunStore) allRuns(id string) (pending Runs, active Runs, archive Runs) {

	r.Lock()
	defer r.Unlock()

	for _, t := range r.pending.Todos {
		if t.Ref.FlowRef.ID != id {
			continue
		}
		pending = append(pending, &Run{
			Ref: t.Ref,
		})
	}

	for _, t := range r.active {
		if t.Ref.FlowRef.ID != id {
			continue
		}
		active = append(active, t)
	}

	// TODO - page or e.g. limit to last 100
	for _, t := range r.archive {
		if t.Ref.FlowRef.ID != id {
			continue
		}
		archive = append(archive, t)
	}

	return pending, active, archive
}
