// Package counter provides a map of counters whos behaviour is defined
// by the Incrementer passed into New
package counter

import (
	"sync"
)

// Incrementer implements an atomic CAS incrementer
type Incrementer interface {
	// Inc increments the named counter only if prev matches existing
	// if it matches then the new value and true is returned
	// if it does not match then the existing value and false is returned and
	Inc(name string, prev int64) (int64, bool)
}

var (
	counters    map[string]int64
	incrementer Incrementer
	mu          sync.RWMutex
)

// New creates new counters using the passed in Incrementer
func New(i Incrementer) {
	mu.Lock()
	defer mu.Unlock()
	counters = map[string]int64{}
	incrementer = i
}

// Inc atomically increments the named counter returning it last known value
// and true if this call was responsible for setting that value
func Inc(name string) (int64, bool) {
	mu.Lock()
	defer mu.Unlock()
	v := counters[name]
	new, ok := incrementer.Inc(name, v)
	counters[name] = new
	return new, ok
}

// IncFunc is function adapter type to be used as an adapter to an Incrementer
type IncFunc func(int64) (int64, bool)

// Inc allows incFunc to be an Incrementer
func (f IncFunc) Inc(name string, prev int64) (int64, bool) {
	return f(prev)
}

// GetSimpleInc return a simple Incrementer for use in a single process
func GetSimpleInc() IncFunc {
	return func(prev int64) (int64, bool) {
		return prev + 1, true
	}
}
