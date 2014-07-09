package main

import (
	"customfloe"
	f "floe/workflow/flow"
	"fmt"
	"time"
)

var project *f.Project

func setup() {
	// load in our specific flows
	project = customflow.GetFlows()
}

func exec(name string, step time.Duration) {

	flow := project.Flows[name]

	// todo fill in initial properties
	props := make(f.Props)

	props["workspace"] = "workspace"
	props["path"] = "/"

	go flow.Exec(props)

	loop := true
	go func() {
		for loop {
			stat := <-flow.CStat
			fmt.Println("          -------------> Status", stat)
		}
		fmt.Println("loop stoppped")
	}()

	go func() {
		for loop {
			time.Sleep(step)
			fmt.Println("          (Stepping)")
			flow.Step(1)
		}
		fmt.Println("stepper loop stoppped")
	}()

	res := <-flow.CEnd

	loop = false

	fmt.Println("end result", res)

	if res.Status == 0 {
		fmt.Println("FLOW SUCCEEDED")
	} else {
		fmt.Println("FLOW FAILED")
	}
}
