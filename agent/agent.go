package agent

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/floe"
	"github.com/floeit/floe/workfloe/par"
)

// thisAgent is the agent for this server - it may be the main ui serving agent or one of the slaves
var thisAgent *Agent

// GetFloesFunc is the function definition for the function that adds floes to a project
type GetFloesFunc func(p *floe.Project, env string)

// Ref describes a particular agent
type Ref struct { // TODO: load this from config
	ID         string
	Name       string
	URL        string // if local then this is empty
	AdminToken string
}

// Agent composes a reference that carries information about this agent and a project
type Agent struct {
	ref     Ref
	project *floe.Project
}

// NewAgent generates a new agent
func NewAgent(id, name string) *Agent {
	id = floe.MakeID(id) // tidy the id
	a := &Agent{
		ref: Ref{
			ID:   id,
			Name: name,
		},
	}
	// add to the list
	thisAgent = a

	return a
}

// SetToken sets our common cluster admin token
func (a *Agent) SetToken(t string) {
	a.ref.AdminToken = t
}

// Setup loads in our specific floes
func (a *Agent) Setup(env string, getFloesFunc GetFloesFunc, wsRoot string) {
	flag.Parse()
	log.Info("Setting up floe for: ", a.ref.ID)
	a.project = floe.NewProject()
	a.project.SetRoot(wsRoot)
	getFloesFunc(a.project, env)
}

// start a particular floe
func (a *Agent) start(floeID string, delay time.Duration, endChan chan *par.Params) (int, error) {
	return a.project.Start(floeID, delay, endChan, hub)
}

// stop any floe in progress
func (a *Agent) stop(floeID string) error {
	return a.project.Stop(floeID)
}

// start the floe and return - expecting some other thing is looking at statuses (e.g. a ajax request)
func (a *Agent) execAsync(floeID string, delay time.Duration) (int, error) {
	rid, err := a.start(floeID, delay, nil)
	return rid, err
}

// Exec start the floe but block on the end chanel waiting for the result
func (a *Agent) Exec(floeID string, delay time.Duration) error {

	log.Info("FLOW EXEC SYNC ", floeID)

	ec := make(chan *par.Params)

	_, err := a.start(floeID, delay, ec)

	if err != nil {
		return err
	}

	res := <-ec

	log.Info("end result", res)

	if res.Status == 0 {
		log.Info("FLOW SUCCEEDED")
	} else {
		log.Info("FLOW FAILED")
	}

	return nil
}

func (a *Ref) webReq(method, sPath string, rq, rp interface{}) (int, string) {
	path := a.URL + rootPath + sPath

	var b []byte
	if rq != nil {
		var err error
		b, err = json.Marshal(rq)
		if err != nil {
			return rErr, "Can't marshal request : " + err.Error()
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(b))
	if err != nil {
		return rErr, "Can't make request : " + err.Error()
	}

	// this is an admin call
	req.Header.Add("X-Floe-Auth", a.AdminToken)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return rErr, "Can't do request : " + err.Error()
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rErr, "Read response failed : " + err.Error()
	}

	if rp != nil {
		err = json.Unmarshal(body, rp)
		if err != nil {
			return rErr, "Failed to unmarshal response : " + err.Error()
		}
	}

	return resp.StatusCode, ""
}
