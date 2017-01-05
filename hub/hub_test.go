package hub

import (
	"testing"

	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

var in = []byte(`
config:
    hosts:
        - name-or.ip.of.other.host 

flows:
    - id: build-project              # the name of this flow
      ver: 1

      subs:                          # external events to subscribe token
        - name: push                 # name of this subscription
          type: git-push             # the type of this trigger
          opts:
            url: blah.blah           # which url to monitor
            
      tasks: 
        - name: checkout             # the name of this node 
          listen: sub.push.good      # the event that triggers this node
          type: git-merge            # the task type 
          good: [0]                  # define what the good statuses are, default [0]
          ignore-fail: false         # if true only emit good
        
        - name: build                
          listen: task.checkout.good    
          type: exec
          opts:
            cmd: "make build"        # the command to execute 

        - name: test1
          listen: task.build.good    
          type: exec                 # execute a command
          opts:
            cmd: "make test"         # the command to execute 

        - name: test2
          listen: task.build.good    
          type: exec                 # execute a command
          opts:
            cmd: "make test 2"       # the command to execute 

        - name: complete
          listen: merge.merge-tests.all
          type: end                 # getting here means the flow was a success
        
      merges:
        - name: merge-tests
          type: all                 # need all wait events to fire 
          wait: [task.test1.good, task.test2.good]
      
`)

func TestHub(t *testing.T) {
	c, _ := config.ParseYAML(in)
	s := store.NewMemStore()
	hub := NewHub("myhost", c, s, &event.Queue{})

	// add an external event
	hub.Notify(event.Event{
		Tag: "git-push",
		Opts: nt.Opts{
			"url": "blah.blah",
		},
	})

	// and confirm the store has an active list
	ac, _ := s.Load(activeKey)
	actives := ac.(Runs)
	if len(actives) != 1 {
		t.Error("wrong number of active runs")
	}
}

type testObs struct {
	ch chan event.Event
}

func (o testObs) Notify(e event.Event) {
	o.ch <- e
}

func TestHubEventQueue(t *testing.T) {
	c, _ := config.ParseYAML(in)
	s := store.NewMemStore()
	q := &event.Queue{}

	NewHub("myhost", c, s, q)

	// register a test observer
	to := testObs{
		ch: make(chan event.Event, 2),
	}
	q.Register(to)

	// add an external event
	pe := event.Event{
		Tag: "git-push",
		Opts: nt.Opts{
			"url":       "blah.blah",
			"from_hash": "from123456",
			"to_hash":   "to7890",
		},
	}
	q.Publish(pe)
	q.Publish(pe)

	// wait for our observer to receive 2 events
	e := <-to.ch
	if e.ID != 1 {
		t.Error("got out of order event")
	}
	e = <-to.ch
	if e.ID != 2 {
		t.Error("got out of order event wanted 2", e.ID)
	}
	if e.Tag != "sub.push.good" {
		t.Error("got bad event", e.Tag)
	}

	time.Sleep(10 * time.Second)

	// and confirm the store has an active list
	ac, _ := s.Load(activeKey)
	actives := ac.(Runs)
	if len(actives) != 1 {
		t.Error("wrong number of active runs")
	}
}
