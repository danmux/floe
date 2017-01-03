package hub

import (
	"sync"

	"github.com/floeit/floe/config"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

const activeKey = "active-list"

// Run describes a specific invocation of a flow
type Run struct {
	Ref event.RunRef
}

// Runs is a list of Run
type Runs []Run

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

func newRunStore(store store.Store) RunStore {
	return RunStore{
		store: store,
	}
}

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

// AddActiveFlow adds the active configs to the active list saves it, and returns the run id
func (r *RunStore) AddActiveFlow(flow config.FlowRef) (int64, error) {
	r.Lock()
	defer r.Unlock()
	id := r.nextID()
	r.Active = append(r.Active, Run{
		Ref: event.RunRef{
			FlowRef: flow,
			ID:      id,
		},
	})

	return id, r.Active.Save(activeKey, r.store)
}
