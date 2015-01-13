package main

// orchestra - the set of orchestration routines
import (
	"errors"
	"flag"
	f "floe/workflow/flow"
	"github.com/golang/glog"
	"time"
)

// the global project
var project *f.Project

type GetFlowsFunc func(env string) *f.Project

// load in our specific flows
func setup(env string, getfloesFunc GetFlowsFunc) {
	flag.Parse()
	glog.Info("Floe starting")
	project = getfloesFunc(env)
	project.RunTriggers()
}

// start a particular flow -
func start(flowId string, delay time.Duration, endChan chan *f.Params) (*f.FlowLauncher, error) {
	launcher, ok := project.FlowLaunchers[flowId]

	if !ok {
		glog.Error("cant start - flow not found ", flowId)
		return nil, errors.New("flow not found")
	}

	glog.Infoln("executing:", flowId)

	go launcher.Start(delay, endChan)

	glog.Infoln("started:", flowId)

	return launcher, nil
}

// stop any flow in progress
func stop(flowId string) error {
	flow, ok := project.FlowLaunchers[flowId]

	if !ok {
		glog.Error("cant stop - flow not found ", flowId)
		return errors.New("flow not found")
	}

	// daleks atack! (and im not even a Dr Who nerd, but I did shit myself as a kid!)
	flow.ExterminateExterminate()

	return nil
}

// start the flow and return - expecting some other thing is looking at statuses (e.g. a ajax request)
func exec_async(flowId string, delay time.Duration) (*f.FlowLauncher, error) {
	flow, err := start(flowId, delay, nil)

	if err != nil {
		return nil, err
	}

	return flow, nil
}

// start the flow but block waiting for the result
func exec(flowId string, delay time.Duration) error {

	ec := make(chan *f.Params)

	_, err := start(flowId, delay, ec)

	if err != nil {
		return err
	}

	res := <-ec

	glog.Infoln("end result", res)

	if res.Status == 0 {
		glog.Infoln("FLOW SUCCEEDED")
	} else {
		glog.Infoln("FLOW FAILED")
	}

	return nil
}
