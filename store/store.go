package store

import (
	"sync"
)

// Store links events to the config rules
type Store interface {
	Save(key string, data interface{}) error
	Load(key string) (interface{}, error)
	// Event(event.Event)
}

type MemStore struct {
	sync.RWMutex
	stuff map[string]interface{}
}

func NewMemStore() *MemStore {
	return &MemStore{
		stuff: map[string]interface{}{},
	}
}

func (m *MemStore) Save(key string, data interface{}) error {
	m.Lock()
	defer m.Unlock()
	m.stuff[key] = data
	return nil
}

func (m *MemStore) Load(key string) (interface{}, error) {
	m.RLock()
	defer m.RUnlock()
	d := m.stuff[key]
	return d, nil
}
