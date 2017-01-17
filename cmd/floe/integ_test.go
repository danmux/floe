package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/store"
	"github.com/floeit/old/log"
)

const adminToken = "a-test-admin-token"

var (
	once sync.Once
	base string
)

func TestWebUnauth(t *testing.T) {
	setup(t)
	resp := &genResp{}

	// auth should fail with a missing session
	ms := "missing session"
	webGet(t, "", "/flows", resp, []int{401}) // unauth
	if resp.Message != ms {
		t.Errorf("should have got `%s` but got: `%s`", ms, resp.Message)
	}

	return

	// auth should fail with an invalid session
	ms = "invalid session"
	webGet(t, "unauth-tok", "/flows", resp, []int{401})
	if resp.Message != ms {
		t.Errorf("should have got `i%s` but got: `%s`", ms, resp.Message)
	}

	// admin token should authenticate
	if !webGet(t, adminToken, "/flows", resp, []int{200}) {
		t.Errorf("admin should have got flows: %s", resp.Message)
	}

	// logged in user should be authenticated
	token := webLogin(t)
	if !webGet(t, token, "/flows", resp, []int{200}) { // authed
		t.Errorf("logged in user should have got flows: %s", resp.Message)
	}
}

func TestWebLaunch(t *testing.T) {
	setup(t)

	tok := webLogin(t)
	flows := &flowsResp{}
	resp := &genResp{
		Payload: flows,
	}

	if !webGet(t, tok, "/flows", resp, []int{200}) { // authed
		t.Error("getting flows failed")
	}
	if len(flows.Flows) != 2 {
		t.Fatal("shoulda got 2 flow, got:", len(flows.Flows))
	}

	// TODO get the form description

	// start a flow from a form
	pl := struct {
		Ref     config.FlowRef
		Answers nt.Opts
	}{
		Ref: config.FlowRef{
			ID:  "build-project",
			Ver: 1,
		},
		Answers: nt.Opts{
			"from_hash": "foooble",
			"to_hash":   "asdfghj",
		},
	}
	if !webPost(t, tok, "/subs/data", pl, nil, []int{200}) {
		t.Error("data trigger failed")
	}

	time.Sleep(time.Second * 2)

	// TODO - get list of jobs

	// wait for jobs to finish

	return // TODO

	//flid := flows.Floes[0].ID

	// p, _ := json.MarshalIndent(flows, "", "  ")

	// if flid != "test-build" {
	// 	t.Fatal("bad floe ID", flid)
	// }

	// time.Sleep(time.Second * 3)

	// resp = &genResp{}
	// if ok := webGet(t, tok, "/flows/"+flid, resp, []int{200}); !ok {
	// 	t.Error("getting floe failed")
	// }

	// p, _ = json.MarshalIndent(resp, "", "  ")
}

func setupWeb(t *testing.T) {
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

      subs:                          # external events to subscribe token
        - name: push                 # name of this subscription
          type: git-push             # the type of this trigger
          opts:
            url: blah.blah           # which url to monitor

        - name: start
          type: data
          opts:
            form:
              title: Start
              fields:
                - id: from_hash
                  prompt: From Branch (or hash)
                  type: string
                - id: to_hash
                  prompt: To Branch (or hash)
                  type: string
      
      merges:
        - name: subs
          type: any
          wait: [sub.push.good, sub.start.good]
            
      tasks: 
        - name: checkout             # the name of this node 
          listen: merge.subs.good    # the event tag that triggers this node
          type: git-merge            # the task type 
          good: [0]                  # define what the good statuses are, default [0]
          ignore-fail: false         # if true only emit good
          use-status: true
        
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

        - name: complete
          listen: task.test.good
          type: end                 # getting here means the flow was a success

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

	log.SetLevel(8)
	basePath := "/build/api"
	addr := "127.0.0.1:3013"
	base = "http://" + addr + basePath

	s := store.NewMemStore()
	sch := make(chan bool)
	go func() {
		err := start("hi1", "%tmp/floe", addr, adminToken, in, s)
		if err != nil {
			t.Error(err)
			sch <- false
		}
	}()

	ready := make(chan bool)
	go func() {
		ready <- waitAPIReady(t)
	}()

	// wait for api decision or start fail
	select {
	case <-sch:
	case res := <-ready:
		if !res {
			t.Fatal("failed to wait or server to come up")
		}
	}

}

func setup(t *testing.T) {
	once.Do(func() {
		setupWeb(t)
	})
}

type genResp struct {
	Message string
	Payload interface{}
}

type summaryStruct struct {
	ID     string
	Name   string
	Status string
}

type flowsResp struct {
	Flows []summaryStruct
}

func webLogin(t *testing.T) string {
	resp := &genResp{}
	v := struct {
		User     string
		Password string
	}{
		User:     "admin",
		Password: "password",
	}

	pl := struct {
		User  string
		Role  string
		Token string
	}{}

	resp.Payload = &pl

	webPost(t, "", "/login", v, resp, []int{200}) // login
	if resp.Message != "OK" {
		t.Errorf("login should have got `OK` but got: `%s`", resp.Message)
	}

	if pl.Token == "" {
		t.Errorf("login should have got none empty token")
	}
	return pl.Token
}

// func webLogout(t *testing.T) {
// 	resp := &genResp{}
// 	v := struct {
// 		User     string
// 		Password string
// 	}{
// 		User:     "admin",
// 		Password: "password",
// 	}

// 	pl := struct {
// 		User  string
// 		Role  string
// 		Token string
// 	}{}

// 	resp.Payload = &pl

// 	webPost(t, "/logout", v, resp, []int{200}) // login
// 	if resp.Message != "OK" {
// 		t.Errorf("login should have got `OK` but got: `%s`", resp.Message)
// 	}

// 	if pl.Token != "" {
// 		t.Errorf("login should have got none empty token")
// 	}
// }

// --- helpers n stuff
func waitAPIReady(t *testing.T) bool {
	for n := 0; n < 10; n++ {
		good := webReq(t, "OPTIONS", "", "/", nil, nil, []int{200}, false)
		if good {
			return true
		}
		time.Sleep(time.Millisecond * 250)
	}
	return false
}

func webGet(t *testing.T, tok, path string, r interface{}, expected []int) bool {
	return webReq(t, "GET", tok, path, nil, r, expected, true)
}

func webPost(t *testing.T, tok, path string, q, r interface{}, expected []int) bool {
	return webReq(t, "POST", tok, path, q, r, expected, true)
}

func webPut(t *testing.T, tok, path string, q, r interface{}, expected []int) bool {
	return webReq(t, "PUT", tok, path, q, r, expected, true)
}

func webReq(t *testing.T, method, tok, spath string, rq, rp interface{}, expected []int, fail bool) bool {
	path := base + spath

	var b []byte
	if rq != nil {
		var err error
		b, err = json.Marshal(rq)
		if err != nil {
			t.Error("Can't marshal request", err)
			rp = nil
			return false
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(b))

	if err != nil {
		panic(err)
	}

	req.Header.Add("X-Floe-Auth", tok)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		if fail {
			t.Error("Get failed", err)
		}
		rp = nil
		return false
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if fail {
			t.Error("Read get response failed", err)
		}
		rp = nil
		return false
	}

	// t.Log(string(body))

	// t.Log("sc", resp.StatusCode)

	codeGood := false
	for _, c := range expected {
		if resp.StatusCode == c {
			codeGood = true
			break
		}
	}

	if !codeGood {
		if fail {
			t.Errorf("Bad response %d, wanted one of %v for [%s:%s]", resp.StatusCode, expected, method, spath)
		}
		rp = nil
	}

	if rp != nil {
		err = json.Unmarshal(body, rp)
		if err != nil {
			if fail {
				t.Error("Failed to unmarshal response", err)
			}
			rp = nil
			return false
		}
	}

	return codeGood
}
