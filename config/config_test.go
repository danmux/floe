package config

import (
	"testing"

	nt "github.com/floeit/floe/config/nodetype"
)

func TestZeroNID(t *testing.T) {

	fix := []struct {
		idIn      string
		nameIn    string
		idOut     string
		nameOut   string
		shouldErr bool
		help      string
	}{
		{
			idIn:      "",
			nameIn:    "",
			shouldErr: true,
			help:      "id and name cant both be blank",
		},
		{
			idIn:      "a.b",
			shouldErr: true,
			help:      "no . allowed in id",
		},
		{
			idIn:      "a b",
			shouldErr: true,
			help:      "no space allowed in id",
		},
		{
			idIn:      "a-b",
			idOut:     "a-b",
			nameOut:   "a b",
			shouldErr: false,
			help:      "name from good id",
		},
		{
			nameIn:    "a b.2",
			nameOut:   "a b.2",
			idOut:     "a-b-2",
			shouldErr: false,
			help:      "id from good name",
		},
		{
			idIn:      "a-b--",
			idOut:     "a-b",
			nameOut:   "a b",
			shouldErr: false,
			help:      "name from id with trailing -",
		},
		{
			idIn:      "--a-b..",
			idOut:     "a-b",
			nameOut:   "a b",
			shouldErr: false,
			help:      "name from id with leading - and trailing .",
		},
	}
	for i, f := range fix {
		fl := &Flow{
			ID:   f.idIn,
			Name: f.nameIn,
		}
		err := zeroNID(fl)

		if !f.shouldErr {
			if err != nil {
				t.Errorf("%d should not have got an error (%s)", i, f.help)
			} else {
				if fl.Name != f.nameOut {
					t.Errorf("%d Name not as expected (%s), wanted (%s), got (%s)", i, f.help, f.nameOut, fl.Name)
				}
				if fl.ID != f.idOut {
					t.Errorf("%d ID not as expected (%s), wanted (%s), got (%s)", i, f.help, f.idOut, fl.ID)
				}
			}
		} else if err == nil {
			t.Errorf("%d should have got an error (%s)", i, f.help)
		}
	}

}

var in = []byte(`
config:
    hosts:
        - name-or.ip.of.other.host 

flows:
    - id: build-project              # the name of this flow
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
            form:
              title: Start
              fields:
                - id: branch
                  prompt: Branch
                  type: string
      
      tasks: 
        - name: checkout             # the name of this node 
          listen: sub.git-push.good  # the event tag that triggers this node
          type: git-merge            # the task type 
          good: [0]                  # define what the good statuses are, default [0]
          ignore-fail: false         # if true only emit good
        
        - name: build                
          listen: task.checkout.good    
          type: exec
          opts:
            cmd: "make build"        # the command to execute 

        - id: test                
          listen: task.build.good    
          type: exec                 # execute a command
          opts:
            cmd: "make test"         # the command to execute 
    
    - id: build-merge
      ver: 1
      
      subs:
        - name: push
          type: git-merge
          opts:
            url: blah.blah
            
      tasks: 
        - name: checkout
          listen: sub.git-push.good
          type: git-checkout
          good: [0]
          ignore-fail: false    
`)

func TestYaml(t *testing.T) {
	c, err := ParseYAML(in)
	if err != nil {
		t.Fatal(err)
	}

	fl := c.Flows[0]
	if !fl.ReuseSpace {
		t.Error("ReuseSpace should be true")
	}
	if len(fl.HostTags) != 3 {
		t.Error("wrong number of host tags", len(fl.HostTags))
	}
	if len(fl.ResourceTags) != 2 {
		t.Error("wrong number of resource tags", len(fl.ResourceTags))
	}

	fns := c.FindFlowsBySubs("data", nil, nt.Opts{"url": "blah.blah"})
	if len(fns) != 1 {
		t.Fatal("did not find the flow based on this sub", len(fns))
	}

	var ff FoundFlow
	for _, ff = range fns {
		break
	}
	ns := ff.Nodes
	if len(ns) != 2 {
		t.Fatal("did not find the nodes based on this sub", len(ns))
	}
	if ns[0].FlowRef().ID != "build-project" {
		t.Error("flow ID not correct", ns[0].FlowRef().ID)
	}
	if ns[0].FlowRef().Ver != 1 {
		t.Error("flow Ver not correct", ns[0].FlowRef().Ver)
	}

	// test finding a node in the known flow
	fr := FlowRef{
		ID:  "build-project",
		Ver: 1,
	}
	fsf, ok := c.FindNodeInFlow(fr, "sub.git-push.good")
	if !ok {
		t.Fatal("could not find flow")
	}
	ns = fsf.Nodes
	if ns[0].NodeRef().Class != NcTask {
		t.Error("got wrong node class")
	}
	if ns[0].NodeRef().Class != NcTask {
		t.Error("got wrong node class", ns[0].NodeRef().Class)
	}
	if ns[0].NodeRef().ID != "checkout" {
		t.Error("got wrong node id", ns[0].NodeRef().ID)
	}
}
