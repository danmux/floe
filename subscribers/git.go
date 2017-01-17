package subscribers

import "github.com/coreos/etcd/store"

// Listener
type Listener struct {
	store store.Store
}
