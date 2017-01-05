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
	activeKey  = "active-list"
	archiveKey = "archive-list"
)

type merge struct {
	Waits map[string]bool // per node id - each wait event received
	Opts  nt.Opts         // merged opts from all listens
}

// Run describes a specific invocation of a flow
type Run struct {
	sync.RWMutex
	Ref       event.RunRef
	StartTime time.Time
	EndTime   time.Time
	Ended     bool

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

func (r *Run) end() {
	r.Lock()
	defer r.Unlock()
	r.EndTime = time.Now()
	r.Ended = true
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

// AddActiveFlow adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) findActiveRun(id int64) (int, *Run) {
	r.RLock()
	defer r.RUnlock()
	for i, run := range r.Active {
		if run.Ref.ID == id {
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

func (r *RunStore) end(run *Run) {
	run.end()

	i, _ := r.findActiveRun(run.Ref.ID)

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

// AddActiveFlow adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) addActiveFlow(flow config.FlowRef, hostID string) (int64, error) {
	r.Lock()
	defer r.Unlock()
	id := r.nextID()
	r.Active = append(r.Active, &Run{
		Ref: event.RunRef{
			FlowRef: flow,
			HostID:  hostID,
			ID:      id,
		},
		StartTime:  time.Now(),
		MergeNodes: map[string]merge{},
	})

	return id, r.Active.Save(activeKey, r.store)
}

// in scope of AddActiveFlow lock
func (r *RunStore) nextID() int64 {
	var max int64
	// if we have actives in play then these must have the max id
	if len(r.Active) > 0 {
		for _, r := range r.Active {
			if r.Ref.ID > max {
				max = r.Ref.ID
			}
		}
		return max + 1
	}
	// otherwise the archives have the max id
	for _, r := range r.Archive {
		if r.Ref.ID > max {
			max = r.Ref.ID
		}
	}
	return max + 1
}
