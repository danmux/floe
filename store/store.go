package store

// Store links events to the config rules
type Store interface {
	Save(key string, data interface{}) error
	Load(key string) (interface{}, error)
	// Event(event.Event)
}

type MemStore struct {
	stuff map[string]interface{}
}

func NewMemStore() *MemStore {
	return &MemStore{
		stuff: map[string]interface{}{},
	}
}

func (m *MemStore) Save(key string, data interface{}) error {
	m.stuff[key] = data
	return nil
}

func (m *MemStore) Load(key string) (interface{}, error) {
	d := m.stuff[key]
	return d, nil
}
