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
		if ws.BasePath != "/foo/bar/spaces/testflow/ws/h1-5" {
			t.Errorf("base path is wrong <%s>", ws.BasePath)
		}
	}
	node := &task{
		exec: exec,
	}
	e := event.Event{}
	run := newRun(&Pend{
		Ref: runRef,
	})
	ws := h.prepareForExec(run.Ref, &e, false, nil)
	h.executeNode(run, node, e, ws)
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
          type: git-checkout         # the task type 
          good: [0]                  # define what the good statuses are, default [0]
          opts:
            url: git@github.com:floeit/floe-test.git
        
        - name: build                
          listen: task.checkout.good    
          type: exec
          opts:
            cmd: "ls /"        # the command to execute 

        - name: test1
          listen: task.build.good    
          type: exec                 # execute a command
          opts:
            cmd: "echo test1"         # the command to execute 

        - name: test2
          listen: task.build.good    
          type: exec                 # execute a command
          opts:
            cmd: "echo test2"       # the command to execute 

        - name: merge-tests
          class: merge
          type: all                 # need all wait events to fire 
          wait: [task.test1.good, task.test2.good]

        - name: complete
          listen: merge.merge-tests.good
          type: end                 # getting here means the flow was a success
`)

func waitEvtTimeout(t *testing.T, ch chan event.Event, msg string) *event.Event {
	select {
	case e := <-ch:
		return &e
	case <-time.After(time.Second * 20):
		t.Fatal("timed out waiting for an event -", msg)
	}
	return nil
}

func TestHubEvents(t *testing.T) {
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
	})

	// get the first events and confirm correct tag order
	expectedTags := []string{
		"inbound.data",
		"sys.state",
		"sys.state",
		"trigger.good",
		"sys.state",
		"sys.node.start",
		"task.checkout.error",
	}
	events := make([]*event.Event, len(expectedTags))
	for i := 0; i < len(events); i++ {
		e := waitEvtTimeout(t, to.ch, "test hub event required list")
		events[e.ID-1] = e
	}

	for i := 0; i < len(events); i++ {
		if events[i].Tag != expectedTags[i] {
			t.Errorf("got %d tag wrong: wanted:%s got:%s", i, expectedTags[i], events[i].Tag)
		}
	}

	// and confirm the store has no runs still pending
	pend := &pending{}
	s.Load(pendingKey, pend)
	if len(pend.Pends) != 0 {
		t.Error("wrong number of pending runs", len(pend.Pends))
	}

	// wait for end of floe event
	e := waitEvtTimeout(t, to.ch, "test hub event sys.end")
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
			"ref": "blah.blah",
		},
	})

	counts := map[string]int{}
	done := make(chan struct{})
	// count all events until the end
	go func() {
		for {
			e := waitEvtTimeout(t, to.ch, "test hub event final end")
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
		"sys.node.update":        18,
		"task.test2.good":        1,
	}
	select {
	case <-time.After(time.Second * 60):
		t.Fatal("timed out")
	case <-done:
		for k, v := range expected {
			if counts[k] < v {
				t.Errorf("got wrong number of events, %s was %d expected %d", k, counts[k], v)
			}
		}
	}
}

var inData = []byte(`
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
            - name: build                
              listen: trigger.good    
              type: exec
              opts:
                cmd: "ls"        # the command to execute 
    
            - name: Sign Off
              type: data
              listen: task.build.good    # for a data node this event has to have occured before the data node can accept data
              opts:
                form:
                  title: Sign off Manual Testing
                  fields:
                    - id: tests_passed
                      prompt: Did the manual testing pass?
                      type: bool
                    - id: to_hash
                      prompt: To Branch (or hash)
                      type: string
    
            - name: test1
              listen: task.sign-off.good    
              type: exec                 # execute a command
              opts:
                cmd: "echo test"         # the command to execute 
    
            - name: complete
              listen: task.test1.good
              type: end                 # getting here means the flow was a success
    `)

func TestHubData(t *testing.T) {
	t.Parallel()

	c, err := config.ParseYAML(inData)
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.NewLocalStore("%tmp")
	if err != nil {
		t.Fatal(err)
	}
	q := &event.Queue{}
	// so we can wait for events to occur after the trigger
	to := &testObs{
		ch: make(chan event.Event, 2),
	}
	q.Register(to)

	// make a new hub
	New("h2", "master", "~/floe", "admintok", c, s, q)

	// start the flow
	// add an external event whose opts dont match those needed by git-merge so will error
	q.Publish(event.Event{
		Tag: "inbound.data", // will match the trigger type
		Opts: nt.Opts{
			"url": "blah.blah",
		},
	})

	// see if we can get the needs data event
	for i := 0; i < 20; i++ {
		e := waitEvtTimeout(t, to.ch, "test hub data required")
		if e.Tag == "sys.data.required" {
			break
		}
	}

	// add an external event whose opts dont match those needed by git-merge so will error
	q.Publish(event.Event{
		Tag: "inbound.data", // will match the data types
		RunRef: event.RunRef{
			FlowRef: config.FlowRef{
				ID:  "build-project",
				Ver: 1,
			},
			Run: event.HostedIDRef{
				HostID: "h2",
				ID:     1,
			},
		},
		SourceNode: config.NodeRef{
			ID: "sign-off",
		},
		Opts: nt.Opts{
			"tests_passed": "true",
			"to_hash":      "blhahaha",
		},
		Good: true,
	})

	// TODO add test for partial data
	// TODO add test for making node bad

	for i := 0; i < 20; i++ {
		e := waitEvtTimeout(t, to.ch, "test hub data sys.end")
		if e.Tag == "sys.end.all" {
			break
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

func TestMergeEnvOpts(t *testing.T) {
	t.Parallel()

	o := nt.Opts{
		"env": []string{"OOF={{ws}}/oof"},
	}
	mergeEnvOpts(o, []string{"DOOF=oops"})
	env := o["env"].([]string)
	if env[0] != "DOOF=oops" {
		t.Error("expand failed", env[0])
	}
	if env[1] != "OOF={{ws}}/oof" {
		t.Error("expand failed", env[1])
	}
}
