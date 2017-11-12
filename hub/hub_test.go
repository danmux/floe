package hub

import (
	"os/user"
	"testing"

	"sync"

	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/store"
)

type task struct {
	exec func(ws nt.Workspace)
}

func (t *task) Execute(ws nt.Workspace, opts nt.Opts) (int, nt.Opts, error) {
	if t.exec != nil {
		t.exec(ws)
	}
	return 0, nil, nil
}

func (t *task) Status(status int) (string, bool) {
	return "good", true
}

func (t *task) FlowRef() config.FlowRef {
	return config.FlowRef{}
}

func (t *task) NodeRef() config.NodeRef {
	return config.NodeRef{}
}

func (t *task) Class() config.NodeClass {
	return config.NcTask
}

func (t *task) TypeOfNode() string {
	return "foo"
}

func (t *task) Waits() int {
	return 0
}

func TestExecuteNode(t *testing.T) {
	h := Hub{
		basePath: "/foo/bar",
		queue:    &event.Queue{},
	}
	runRef := &event.RunRef{
		FlowRef: config.FlowRef{
			ID: "testflow",
		},
		Run: event.HostedIDRef{
			HostID: "h1",
			ID:     5,
		},
	}
	didExec := false
	exec := func(ws nt.Workspace) {
		didExec = true
		if ws.BasePath != "/foo/bar/testflow/ws/h1-5" {
			t.Errorf("base path is wrong <%s>", ws.BasePath)
		}
	}
	node := &task{
		exec: exec,
	}
	e := event.Event{}
	h.executeNode(runRef, node, e, false)
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
          listen: trigger.good       # the event that triggers this node
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
          listen: merge.merge-tests.good
          type: end                 # getting here means the flow was a success
        
      merges:
        - name: merge-tests
          type: all                 # need all wait events to fire 
          wait: [task.test1.good, task.test2.good]
      
`)

func TestHub(t *testing.T) {
	c, _ := config.ParseYAML(in)
	s := store.NewMemStore()
	q := &event.Queue{}

	// make a new hub
	New("h1", "master", "~/floe", "admintok", c, s, q)

	to := &testObs{
		ch: make(chan event.Event, 2),
	}
	q.Register(to)

	// add an external event whose opts dont match those needed by git-merge so will error
	q.Publish(event.Event{
		Tag: "data", // will match the trigger type
		Opts: nt.Opts{
			"url": "blah.blah",
		},
	})

	// sink the publish event
	<-to.ch

	// grab the trigger event
	e := <-to.ch
	if e.Tag != "trigger.good" {
		t.Error("got bad event", e.Tag)
	}

	// and confirm the store has a run pending
	pd, _ := s.Load(pendingKey)
	pend := pd.(pending)
	if len(pend.Todos) != 1 {
		t.Error("wrong number of pending runs", len(pend.Todos))
	}

	// record the error event
	e = <-to.ch

	// wait for end of floe event
	<-to.ch

	spew.Dump(e)
	if e.Good {
		t.Error("flow should have ended badly")
	}

	// and confirm the store has a no runs pending
	pd, _ = s.Load(pendingKey)
	pend = pd.(pending)
	if len(pend.Todos) != 0 {
		t.Error("wrong number of pending runs after finishing", len(pend.Todos))
	}

	// add an external event whose do match those needed by git-merge so will execute
	q.Publish(event.Event{
		Tag: "data", // will match the trigger type
		Opts: nt.Opts{
			"from_hash": "blah.blah",
			"to_hash":   "blah.blah",
		},
	})

	// get all events
	for {
		e = <-to.ch

		if e.Tag == "sys.end.all" {
			return
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

func TestEventQueue(t *testing.T) {
	c, _ := config.ParseYAML(in)
	s := store.NewMemStore()
	q := &event.Queue{}

	New("h1", "master", "%tmp/flow", "admin-tok", c, s, q)

	// register a test observer
	to := &testObs{
		ch: make(chan event.Event, 2),
	}
	q.Register(to)

	// add an external event
	pe := event.Event{
		Tag: "data",
		Opts: nt.Opts{
			"url":       "blah.blah",
			"from_hash": "from123456",
			"to_hash":   "to7890",
		},
	}
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
	if e.Tag != "trigger.good" {
		t.Error("got bad event", e.Tag)
	}

	// and confirm the store has an active list
	ac, _ := s.Load(activeKey)
	actives := ac.(Runs)
	if len(actives) != 1 {
		t.Error("wrong number of active runs")
	}

	// wait for end to close the chanel
	for e := range to.ch {
		if e.Tag == "sys.end.all" {
			break
		}
	}

	// and confirm the store has an active list
	ac, _ = s.Load(activeKey)
	actives = ac.(Runs)
	if len(actives) != 0 {
		t.Error("wrong number of active runs")
	}
	// and one archive
	ac, _ = s.Load(archiveKey)
	archives := ac.(Runs)
	if len(archives) != 1 {
		t.Error("wrong number of archives runs")
	}
}

func TestExpandPath(t *testing.T) {
	usr, _ := user.Current()
	hd := usr.HomeDir

	fix := []struct {
		in  string
		out string
		e   bool
	}{
		{in: "~/", out: "", e: true},      // too short
		{in: "~", out: "", e: true},       // too short
		{in: "~/test", out: hd + "/test"}, // sub ~
		{in: "/test/~", out: "/test/~"},   // dont ~
		{in: "test/foo", out: "test/foo"},
		{in: "/test/foo", out: "/test/foo"},
	}
	for i, f := range fix {
		ep, err := expandPath(f.in)
		if (err == nil && f.e) || (err != nil && !f.e) {
			t.Errorf("test %d expected error mismatch", i)
		}
		if ep != f.out {
			t.Errorf("test %d failed, wanted: %s got: %s", i, f.out, ep)
		}
	}

	ep, _ := expandPath("%tmp/test/bar")
	fpos := strings.Index(ep, "/floe")
	if fpos < 5 {
		t.Error("tmp expansion failed", ep)
	}
	if strings.Index(ep, "/test/bar") < fpos {
		t.Error("tmp expansion prefix... isnt ", ep)
	}
}
