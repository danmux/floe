// Package listener listens for events or changes in state in the outside world
package listener

import "github.com/coreos/etcd/store"

// Listener 
type Listener struct {
	store store.Store
}
