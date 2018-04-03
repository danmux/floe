// Package client provides helper functions to communicate with flow servers.
// Each server uses this client to talk to each other, and the client can be used for.
// integration testing, or building other go based interfaces.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

// HostConfig the public config data of a host
type HostConfig struct {
	HostID  string
	BaseURL string
	Online  bool
	Tags    []string
}

// TagsMatch returns true is all tags are present in the receivers tags
func (h HostConfig) TagsMatch(tags []string) bool {
	for _, t := range tags {
		found := false
		for _, ht := range h.Tags {
			if t == ht {
				found = true
				continue
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// FloeHost provides methods to access a host api
type FloeHost struct {
	sync.RWMutex

	// Config is the public config
	config HostConfig

	token string
}

// New returns a new FloeHost
func New(base, token string) *FloeHost {
	fh := &FloeHost{
		config: HostConfig{
			BaseURL: base + "/p2p",
		},
		token: token,
	}
	// start the ping heartbeat to the target floe host
	go fh.pinger()
	return fh
}

// GetConfig returns the config
func (f *FloeHost) GetConfig() HostConfig {
	f.RLock()
	defer f.RUnlock()
	return f.config
}

// AttemptExecute tries to execute the flow matching the flowref and instigating event.
// Returns true if the host accepted the run.
func (f *FloeHost) AttemptExecute(ref event.RunRef, ie event.Event) bool {
	w := wrap{}

	pend := struct {
		Ref             event.RunRef
		InitiatingEvent event.Event
	}{
		Ref:             ref,
		InitiatingEvent: ie,
	}

	code, err := f.post("/flows/exec", pend, &w)
	if err != nil {
		log.Error(err)
		return false
	}
	switch code {
	case http.StatusOK:
		return true
	case http.StatusConflict:
		log.Debugf("host %s busy: %s", f.GetConfig().HostID, w.Message)
	default:
		log.Errorf("got response: %d from %s, with: %s", code, f.GetConfig().HostID, w.Message)
	}

	return false
}

// RunSummaries holds slices of RunSummary for each group of run
type RunSummaries struct {
	Active  []RunSummary
	Pending []RunSummary
	Archive []RunSummary
}

// Append adds the summaries o to the reciever s
func (s *RunSummaries) Append(o *RunSummaries) {
	if o == nil {
		return
	}
	s.Active = append(s.Active, o.Active...)
	s.Pending = append(s.Pending, o.Pending...)
	s.Archive = append(s.Archive, o.Archive...)
}

// RunSummary represents the state of a run
type RunSummary struct {
	Ref       event.RunRef
	ExecHost  string // the id of the host who's actually executing this run
	StartTime time.Time
	EndTime   time.Time
	Ended     bool
	Status    string
	Good      bool
}

// GetRuns - gets the runs from a host for the given id or nil if there is a problem
func (f *FloeHost) GetRuns(id string) *RunSummaries {
	w := wrap{}
	runs := &RunSummaries{}
	w.Payload = runs

	code, err := f.get(fmt.Sprintf("/flows/%s/runs", id), &w)
	if err != nil {
		log.Error(err)
		return nil
	}
	switch code {
	case http.StatusOK:
		return runs
	default:
		log.Errorf("got response: %d from %s, with: %s", code, f.GetConfig().HostID, w.Message)
	}

	return nil
}

// a merge record is kept per node id
type merge struct {
	Waits map[string]bool // each wait event received
	Opts  nt.Opts         // merged opts from all events
}

type data struct {
	Enabled bool    // Enabled is true if the enabling event has occurred
	Opts    nt.Opts // opts from the data event
}

type exec struct {
	Opts nt.Opts  // opts from the exec event
	Logs []string // any output of the node
}

// Run is a specific invocation of a flow
type Run struct {
	Ref        event.RunRef
	ExecHost   string
	StartTime  time.Time
	EndTime    time.Time
	Ended      bool
	Status     string
	Good       bool
	Initiating event.Event
	MergeNodes map[string]merge
	DataNodes  map[string]data
	ExecNodes  map[string]exec
}

// FindRun - finds the run in any of the peer hosts
func (f *FloeHost) FindRun(flowID, runID string) *Run {
	w := wrap{}
	run := &Run{}
	w.Payload = run

	code, err := f.get(fmt.Sprintf("/flows/%s/runs/%s", flowID, runID), &w)
	if err != nil {
		log.Error(err)
		return nil
	}
	switch code {
	case http.StatusOK:
		return run
	case http.StatusNotFound:
	default:
		log.Errorf("got find run response: %d from %s, with: %s", code, f.GetConfig().HostID, w.Message)
	}

	return nil
}

type wrap struct {
	Message string
	Payload interface{}
}

func (f *FloeHost) pinger() {
	tk := time.NewTicker(time.Second * 10)
	for range tk.C {
		baseURL := f.config.BaseURL
		conf, err := f.fetchConf()
		f.Lock()
		if conf.HostID == "" || err != nil {
			log.Error("cant get config from", f.config.BaseURL, err)
			f.config.Online = false
		} else {
			f.config = conf
			f.config.Online = true
			f.config.BaseURL = baseURL
		}
		f.Unlock()
	}
}

func (f *FloeHost) fetchConf() (HostConfig, error) {
	w := wrap{}
	c := struct {
		Config HostConfig
	}{}
	w.Payload = &c
	code, err := f.get("/config", &w)
	if err != nil {
		return c.Config, err
	}
	if code == http.StatusOK {
		return c.Config, nil
	}
	return c.Config, nil
}

func (f *FloeHost) get(path string, r interface{}) (int, error) {
	return f.req("GET", path, nil, r)
}

func (f *FloeHost) post(path string, q, r interface{}) (int, error) {
	return f.req("POST", path, q, r)
}

func (f *FloeHost) put(path string, q, r interface{}) (int, error) {
	return f.req("PUT", path, q, r)
}

func (f *FloeHost) req(method, sPath string, rq, rp interface{}) (status int, err error) {
	f.RLock()
	path := f.config.BaseURL + sPath
	f.RUnlock()

	var b []byte
	if rq != nil {
		b, err = json.Marshal(rq)
		if err != nil {
			return 0, err
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(b))
	if err != nil {
		return 0, err
	}

	// add the auth
	req.Header.Add("X-Floe-Auth", f.token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if rp != nil {
		err = json.Unmarshal(body, rp)
		if err != nil {
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, nil
}
