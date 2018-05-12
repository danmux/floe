package config

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

var flow = &Flow{
	Name: "flow-1",
	Triggers: []*node{
		&node{
			Name: "Start",
		},
	},
	Tasks: []*node{
		&node{
			Name:   "First Task",
			Listen: "trigger.good",
		},
		&node{
			Name:  "Merge Thing",
			Class: "merge",
			Wait:  []string{"task.first.good"},
		},
	},
}

func TestZero(t *testing.T) {
	err := flow.zero()
	if err != nil {
		t.Error(err)
	}

	if flow.Triggers[0].ID != "start" {
		t.Error("zero failed", flow.Triggers[0].ID)
	}

	if flow.Tasks[0].ID != "first-task" {
		t.Error("zero failed", flow.Tasks[0].ID)
	}
	if flow.Tasks[0].Class != NcTask {
		t.Error("zero failed", flow.Tasks[0].Class)
	}
	if flow.Tasks[1].Class != NcMerge {
		t.Error("zero failed on merge node", flow.Tasks[1].Class)
	}
}

func TestMatchTag(t *testing.T) {
	err := flow.zero()
	if err != nil {
		t.Error(err)
	}

	matches := flow.MatchTag("trigger.good")
	if len(matches) != 1 {
		t.Error("did not find task node")
	}

	matches = flow.MatchTag("task.first.good")
	if len(matches) != 1 {
		t.Error("did not find merge node")
	}
}

func TestGetRefType(t *testing.T) {
	t.Parallel()

	fxs := []struct {
		ref string
		typ string
	}{
		{
			ref: "/foo/bar/fl.yaml",
			typ: "local",
		},
		{
			ref: "git@github.com:floeit/floe.git/build/FLOE.yaml",
			typ: "git",
		},
		{
			ref: "http://foo/bar/ml.yaml",
			typ: "web",
		},
		{
			ref: "https://foo/bar/ml.yaml",
			typ: "web",
		},
	}

	for i, fx := range fxs {
		typ := getRefType(fx.ref)
		if typ != fx.typ {
			t.Errorf("%d - got wrong type wanted: <%s>, got: <%s>, for file: <%s>cent2cent", i, fx.typ, typ, fx.ref)
		}
	}
}

// N.B. note spaces for indentation - tabs not allowed
var floeIn = `
id: build-project              # the name of this flow
ver: 1
reuse-space: true               # reuse the workspace (false) - if true /single used 
resource-tags: [couchbase, nic] # resource labels that any other flows cant share
host-tags: [linux, go, couch]   # all these tags must match the tags on any host for it to be able to run there

triggers:                      # external events to subscribe token
  - name: input                # name of this subscription
    type: data                 # the type of this trigger
    opts:
      url: blah.blah           # which url to monitor

  - name: start
    type: data
    opts:
      Form:
        Title: Start
        Fields:
          - Id: branch
            Prompt: Branch
            Type: string

tasks: 
  - name: checkout             # the name of this node 
    listen: trigger.good       # the event tag that fires node (all triggers fire trigger.good)
    type: git-merge            # the task type 
    good: [0]                  # define what the good statuses are, default [0]
    ignore-fail: false         # if true only emit good
  
  - name: build                
    listen: task.checkout.good    
    type: exec
    opts:
      cmd: "make build"        # the command to execute 

  - name: build-osx
    listen: task.checkout.good
    type: exec
    opts:
      cmd: "make build"        # the command to execute 

  - id: builds
    class: merge
    type: all                  # wait for all events
    wait: [task.build.good, task.build-osx.good]
    
  - id: test
    listen: merge.builds.good    
    type: exec                 # execute a command
    opts:
      cmd: "make test"         # the command to execute 

  - name: Complete
    listen: task.test.good
    type: end                # getting here means the flow was a success 'end' is the special definitive end event
`

func TestLoad(t *testing.T) {
	t.Parallel()

	// create the local file
	tf, err := ioutil.TempFile("", "flow-file")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tf.WriteString(floeIn)
	if err != nil {
		t.Fatal(err)
	}
	tf.Close()

	// launch a test server serving the same content
	portChan := make(chan int)
	go func() {
		serveFiles(portChan)
	}()
	port := <-portChan

	// store dl files in this temp cache folder
	tmpCache, err := ioutil.TempDir("", "floe-tests")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{tf.Name(), fmt.Sprintf("http://127.0.0.1:%d/get-file.txt", port)} {
		f := &Flow{
			FlowFile: name,
		}
		err = f.Load(tmpCache)
		if err != nil {
			t.Fatal(err)
		}

		tag := "merge.builds.good"
		ns := f.MatchTag(tag)
		if len(ns) != 1 {
			t.Error("could not find single node", name, tag, len(ns))
		}
		if f.ResourceTags[0] != "couchbase" {
			t.Error("got bad resource tag", name, f.ResourceTags[0])
		}
		if f.ResourceTags[1] != "nic" {
			t.Error("got bad resource tag", name, f.ResourceTags[1])
		}
	}
}

// simple local server that returns a bit of content
func serveFiles(portChan chan int) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/get-file.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(floeIn))
	})
	portChan <- listener.Addr().(*net.TCPAddr).Port
	http.Serve(listener, mux)
}
