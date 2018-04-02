package hub

import (
	"testing"
	"time"

	"sync"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

type task struct {
	exec func(ws *nt.Workspace, updates chan string)
}

func (t *task) NodeRef() config.NodeRef {
	return config.NodeRef{}
}

func (t *task) Status(status int) (string, bool) {
	return config.SubTagBad, true
}

func (t *task) GetTag(string) string {
	return "tag"
}

func (t *task) Execute(ws *nt.Workspace, opts nt.Opts, updates chan string) (int, nt.Opts, error) {
	if t.exec != nil {
		t.exec(ws, updates)
	}
	return 0, nil, nil
}

func TestExecuteNode(t *testing.T) {
	s := store.NewMemStore()
	h := Hub{
		basePath: "/foo/bar",
		queue:    &event.Queue{},
		runs:     newRunStore(s),
	}
	runRef := event.RunRef{
		FlowRef: config.FlowRef{
			ID: "testflow",
		},
		Run: event.HostedIDRef{
			HostID: "h1",
			ID:     5,
		},
	}
	didExec := false
	exec := func(ws *nt.Workspace, updates chan string) {
		didExec = true
		if ws.BasePath != "/foo/bar/testflow/ws/h1-5" {
			t.Errorf("base path is wrong <%s>", ws.BasePath)
		}
	}
	node := &task{
		exec: exec,
	}
	e := event.Event{}
	run := newRun(runRef)
	h.executeNode(run, node, e, false)
	if !didExec {
		t.Error("did not execute executor")
	}
}

var in = []byte(`
common:
    base-url: "/build/api" 
    
flows:
    - id: build-project              # the name of this flow
      ver: 1

      triggers:                      # external events to trigger the flow
        - name: form                 # name of this subscription
          type: data                 # the type of this trigger
          opts:
            url: blah.blah           # which url to monitor
            
      tasks: 
        - name: checkout             # the name of this node 
          listen: trigger.good   # the event that triggers this node
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

        - name: merge-tests
          class: merge
          type: all                 # need all wait events to fire 
          wait: [task.test1.good, task.test2.good]

        - name: complete
          listen: merge.merge-tests.good
          type: end                 # getting here means the flow was a success
`)

func waitEvtTimeout(t *testing.T, ch chan event.Event) *event.Event {
	select {
	case e := <-ch:
		return &e
	case <-time.After(time.Second * 5):
		t.Fatal("timed out waiting for edn event")
	}
	return nil
}

func TestHub(t *testing.T) {
	t.Parallel()

	c, _ := config.ParseYAML(in)
	s, err := store.NewLocalStore("%tmp")
	if err != nil {
		t.Fatal(err)
	}
	q := &event.Queue{}

	// make a new hub
	New("h1", "master", "~/floe", "admintok", c, s, q)

	to := &testObs{
		ch: make(chan event.Event, 2),
	}
	q.Register(to)

	// add an external event whose opts dont match those needed by git-merge so will error
	q.Publish(event.Event{
		Tag: "inbound.data", // will match the trigger type
		Opts: nt.Opts{
			"url": "blah.blah",
		},
	})

	// get the first 6 events and confirm correct tag order
	var events [6]*event.Event
	for i := 0; i < len(events); i++ {
		e := waitEvtTimeout(t, to.ch)
		events[e.ID-1] = e
	}

	tags := []string{
		"inbound.data",
		"sys.state",
		"sys.state",
		"trigger.good",
		"sys.state",
		"task.checkout.error",
	}

	for i := 0; i < len(events); i++ {
		// fmt.Println(events[i].Opts["action"])
		if events[i].Tag != tags[i] {
			t.Errorf("got %d tag wrong: wanted:%s got:%s", i, tags[i], events[i].Tag)
		}
	}

	// and confirm the store has no runs still pending
	pend := &pending{}
	s.Load(pendingKey, pend)
	if len(pend.Pends) != 0 {
		t.Error("wrong number of pending runs", len(pend.Pends))
	}

	// wait for end of floe event
	e := waitEvtTimeout(t, to.ch)
	if e.Tag != "sys.end.all" {
		t.Fatal("should have got the end event", e.Tag)
	}
	if e.Good {
		t.Error("flow should have ended badly")
	}

	// and confirm the store has a no runs active
	act := Runs{}
	s.Load(activeKey, &act)
	if len(act) != 0 {
		t.Error("wrong number of active runs after finishing", len(act))
	}

	// relaunch the flow with an external event whose optsdo match those
	// needed by git-merge so will execute
	q.Publish(event.Event{
		Tag: "inbound.data", // will match the trigger type
		Opts: nt.Opts{
			"from_hash": "blah.blah",
			"to_hash":   "blah.blah",
		},
	})

	counts := map[string]int{}
	done := make(chan struct{})
	// count all events until the end
	go func() {
		for {
			e := waitEvtTimeout(t, to.ch)
			counts[e.Tag] = counts[e.Tag] + 1
			if e.Tag == "sys.end.all" {
				done <- struct{}{}
			}
		}
	}()

	expected := map[string]int{
		"sys.state":              3,
		"task.build.good":        1,
		"task.test1.good":        1,
		"merge.merge-tests.good": 1,
		"sys.end.all":            1,
		"inbound.data":           1,
		"trigger.good":           1,
		"task.checkout.good":     1,
		"sys.node.update":        15,
		"task.test2.good":        1,
	}
	select {
	case <-time.After(time.Second * 60):
		t.Fatal("timed out")
	case <-done:
		for k, v := range expected {
			if counts[k] != v {
				t.Errorf("got wrong number of events, %s was %d expected %d", k, counts[k], v)
			}
		}
	}
}

type testObs struct {
	sync.Mutex
	ch chan event.Event
}

func (o *testObs) Notify(e event.Event) {
	o.ch <- e
}
