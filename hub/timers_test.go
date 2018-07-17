package hub

import (
	"testing"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

type obs func(e event.Event)

func (o obs) Notify(e event.Event) {
	o(e)
}

func TestTimers(t *testing.T) {
	t.Parallel()

	s, err := store.NewLocalStore("%tmp")
	if err != nil {
		t.Fatal(err)
	}

	q := &event.Queue{}

	// make an observer that signals a chanel
	got := make(chan bool, 1)
	f := func(e event.Event) {
		if e.Tag == "sys.state" {
			got <- true
		}
	}
	q.Register(obs(f))

	ts := newTimers(&Hub{
		queue: q,
		runs:  newRunStore(s),
		config: &config.Config{
			Flows: []*config.Flow{
				&config.Flow{
					ID:  "test-flow",
					Ver: 1,
				},
			},
		},
	})

	ts.register(config.FlowRef{
		ID:  "test-flow",
		Ver: 1,
	}, "test-node", nt.Opts{
		"period": 1,
	})

	select {
	case <-time.After(time.Second * 2):
		t.Fatal("no event")
	case <-got:
	}
}
