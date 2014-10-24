package main

// orchestra - the set of orchestration routines
import (
	"customfloe"
	"errors"
	"flag"
	f "floe/workflow/flow"
	"fmt"
	"third_party/github.com/golang/glog"
	"time"
)

var project *f.Project

// load in our specific flows
func setup() {
	flag.Parse()
	glog.Info("Floe starting")
	project = customflow.GetFlows()
}

func start(flowId string) (*f.FlowLauncher, error) {
	flow, ok := project.Flows[flowId]

	if !ok {
		glog.Error("cant start - flow not found ", flowId)
		return nil, errors.New("flow not found")
	}

	props := make(f.Props)

	props[f.KEY_WORKSPACE] = "workspace"
	props["path"] = "/"

	fmt.Println("executing")

	go flow.Exec(props)

	fmt.Println("started")

	return flow, nil
}

func stop(flowId string) error {
	flow, ok := project.Flows[flowId]

	if !ok {
		glog.Error("cant stop - flow not found ", flowId)
		return errors.New("flow not found")
	}

	flow.ExterminateExterminate()

	return nil
}

func exec_async(flowId string, delay time.Duration) (*f.FlowLauncher, error) {
	flow, err := start(flowId)

	if err != nil {
		return nil, err
	}

	go flow.AutoStep(delay, nil)

	return flow, nil
}

func exec(flowId string, delay time.Duration) error {

	flow, err := start(flowId)

	if err != nil {
		return err
	}

	ec := make(chan *f.Params)

	go flow.AutoStep(delay, ec)

	res := <-ec

	fmt.Println("end result", res)

	if res.Status == 0 {
		fmt.Println("FLOW SUCCEEDED")
	} else {
		fmt.Println("FLOW FAILED")
	}

	return nil
}
