package hub

import (
	"sync"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/store"
)

const (
	pendingKey = "pending-list"
	activeKey  = "active-list"
	archiveKey = "archive-list"
)

// Pend is a triggered flow that is waiting for a slave
type Pend struct {
	Ref             event.RunRef
	InitiatingEvent event.Event
}

func (t Pend) String() string {
	return t.Ref.String()
}

func (t Pend) equal(u Pend) bool {
	return t.Ref.Equal(u.Ref)
}

// a merge record is kept per node id
type merge struct {
	Waits map[string]bool // each wait event received
	Opts  nt.Opts         // merged opts from all events
}

type data struct {
	Enabled bool      // Enabled is true if the enabling event has occurred
	Started time.Time // when it became enabled for data
	Stopped time.Time // when data was fully entered
	Opts    nt.Opts   // opts from the data event
}

type exec struct {
	Started time.Time
	Stopped time.Time
	Good    bool     // only valid when Status="finished"
	Opts    nt.Opts  // opts from the exec event
	Logs    []string // any output of the node
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
	Initiating event.Event      // the event that started it all
	MergeNodes map[string]merge // the states of the merge nodes by node id
	DataNodes  map[string]data  // the sates of any data nodes
	ExecNodes  map[string]exec  // the sates of any exec nodes
}

func newRun(pend *Pend) *Run {
	return &Run{
		Ref:        pend.Ref,
		StartTime:  time.Now(),
		Initiating: pend.InitiatingEvent,
		MergeNodes: map[string]merge{},
		DataNodes:  map[string]data{},
		ExecNodes:  map[string]exec{},
	}
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

// updateExecNode adds the output line to the log lines for the nod in this run
func (r *Run) updateExecNode(nodeID string, start, end time.Time, good bool, line string) {
	r.Lock()
	defer r.Unlock()
	m, ok := r.ExecNodes[nodeID]
	if !ok {
		m = exec{}
	}

	if !start.IsZero() {
		m.Started = start.UTC()
	}
	if !end.IsZero() {
		m.Stopped = end.UTC()
		m.Good = good
	}

	if line != "" {
		m.Logs = append(m.Logs, line)
	}
	r.ExecNodes[nodeID] = m
}

// updateDataNode adds the opts form description
func (r *Run) updateDataNode(nodeID string, opts nt.Opts) {
	r.Lock()
	defer r.Unlock()
	m, ok := r.DataNodes[nodeID]
	if !ok {
		m = data{}
	}
	m.Opts = opts
	m.Enabled = true
	m.Started = time.Now() // TODO move this to the hub - when we can handle data input in the run
	r.DataNodes[nodeID] = m
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
	Pends   []*Pend
}

// Save saves the pending list
func (r pending) Save(key string, s store.Store) error {
	return s.Save(key, r)
}

// Load loads the pending list
func (r *pending) Load(key string, s store.Store) error {
	return s.Load(key, r)
}

// Runs is a list of Run
type Runs []*Run

// Save saves the runs
func (r Runs) Save(key string, s store.Store) error {
	return s.Save(key, r)
}

// Load loads the runs
func (r *Runs) Load(key string, s store.Store) error {
	return s.Load(key, r)
}

func (r Runs) find(flowID, runID string) *Run {
	for _, run := range r {
		if run.Ref.FlowRef.ID == flowID && run.Ref.Run.String() == runID {
			return run
		}
	}
	return nil
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
	r := &RunStore{
		store: store,
	}
	// load in any lists
	if err := r.pending.Load(pendingKey, r.store); err != nil {
		log.Error("can not load pending list", err)
	}
	if err := r.active.Load(activeKey, r.store); err != nil {
		log.Error("can not load active list", err)
	}
	if err := r.archive.Load(archiveKey, r.store); err != nil {
		log.Error("can not load archive list", err)
	}

	return r
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
	if err := r.active.Save(activeKey, r.store); err != nil {
		log.Error("could not save", activeKey, err)
	}
	return i, o
}

// TODO - consider buffering these writes if the updates come in fast
func (r *RunStore) updateExecNode(run *Run, nodeID string, start, end time.Time, good bool, line string) {
	run.updateExecNode(nodeID, start, end, good, line)
	r.Lock()
	defer r.Unlock()
	if err := r.active.Save(activeKey, r.store); err != nil {
		log.Error("could not save exe update", activeKey, err)
	}
}

func (r *RunStore) updateDataNode(run *Run, nodeID string, opts nt.Opts) {
	run.updateDataNode(nodeID, opts)
	r.Lock()
	defer r.Unlock()
	if err := r.active.Save(activeKey, r.store); err != nil {
		log.Error("could not save data node", activeKey, err)
	}
}

// end moves the run from active to archive. As a run may have many events that would end it
// only the first one does the others are ignored. Only the ending run returns true.
func (r *RunStore) end(run *Run, status string, good bool) bool {
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

	if err := r.active.Save(activeKey, r.store); err != nil {
		log.Error("could not save", activeKey, err)
	}
	if err := r.archive.Save(archiveKey, r.store); err != nil {
		log.Error("could not save", archiveKey, err)
	}

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
	t := &Pend{
		Ref: event.RunRef{
			FlowRef: flow,
			Run:     run,
		},
		InitiatingEvent: e,
	}
	r.pending.Pends = append(r.pending.Pends, t)

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

// activate adds the active configs to the active list, saves it, and returns the run id
func (r *RunStore) activate(pend *Pend, hostID string) error {
	r.Lock()
	defer r.Unlock()

	// update the runref with this executing host
	pend.Ref.ExecHost = hostID

	r.active = append(r.active, newRun(pend))

	return r.active.Save(activeKey, r.store)
}

func (r *RunStore) allPends() []Pend {
	r.Lock()
	defer r.Unlock()
	t := make([]Pend, len(r.pending.Pends))
	for i, pend := range r.pending.Pends {
		t[i] = *pend
	}
	return t
}

// removePend returns true if the given pending run is removed from the pending list
func (r *RunStore) removePend(pend Pend) (bool, error) {
	r.Lock()
	defer r.Unlock()

	for i, td := range r.pending.Pends {
		if td.equal(pend) {
			// slide them down
			copy(r.pending.Pends[i:], r.pending.Pends[i+1:])
			// explicitly drop the reference to the one left at the end
			r.pending.Pends[len(r.pending.Pends)-1] = nil
			// and remove it from the slice
			r.pending.Pends = r.pending.Pends[:len(r.pending.Pends)-1]

			// save the whole pending list
			return true, r.pending.Save(pendingKey, r.store)
		}
	}
	// If the pend is not found then there is nothing to worry about
	// it is already removed
	return false, nil
}

// find finds the run given by flowID and runID if it exists in the pending, active, or archive runs
func (r *RunStore) find(flowID, runID string) *Run {
	pending := r.pendToRuns(flowID)

	r.Lock()
	defer r.Unlock()

	for _, runs := range []Runs{pending, r.active, r.archive} {
		run := runs.find(flowID, runID)
		if run != nil {
			return run
		}
	}
	return nil
}

func (r *RunStore) pendToRuns(id string) (pending Runs) {
	r.Lock()
	defer r.Unlock()

	for _, t := range r.pending.Pends {
		if t.Ref.FlowRef.ID != id {
			continue
		}
		pending = append(pending, &Run{
			Ref:        t.Ref,
			Initiating: t.InitiatingEvent,
		})
	}
	return pending
}

func (r *RunStore) allRuns(id string) (pending Runs, active Runs, archive Runs) {
	pending = r.pendToRuns(id)

	r.Lock()
	defer r.Unlock()

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
