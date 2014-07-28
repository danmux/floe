package flow

import (
	"fmt"
	"sync"
	"time"
)

type GetFlowFunc func(threadId int) *Workflow

// the end user returns these
type LaunchableFlow interface {
	FlowFunc(threadId int) *Workflow
	Name() string
	Id() string
}

type BaseFlow struct {
	name string
	id   string
}

func (b *BaseFlow) Init(name string) {
	b.name = name
	b.id = MakeID(name)
}

func (b *BaseFlow) Name() string {
	return b.name
}

func (b *BaseFlow) Id() string {
	return b.id
}

// a load of structures to service a multiple workflow threads- which is passed in via the GetFlowFunc which constructs the workflow
// these can be used to fire off parallel test workflows for example - as a load test
type FlowLauncher struct {
	Name          string
	Id            string
	FlowFunc      GetFlowFunc
	Threads       int
	Flows         []*Workflow // each thread creates a full workflow in memory - so the implementor of tasks does not have to wory about thread conflicts
	Props         *Props
	CStat         chan *Params
	iEnd          chan *Params      // internal end chanel for auto stepper
	LastRunResult *FlowLaunchResult // a set of response stats by task id in our workflow for the last run
	// TODO - historical stats / logs
}

func MakeFlowLauncher(launchable LaunchableFlow, threads int) *FlowLauncher {

	return &FlowLauncher{
		Name:     launchable.Name(),
		Id:       launchable.Id(),
		FlowFunc: launchable.FlowFunc,
		Threads:  threads,
	}
}

// the launchers trigger is the end tasknodes trigger
func (fl *FlowLauncher) Trigger() chan *Params {
	return fl.iEnd
}

// can call own step - perhaps via ui
func (fl *FlowLauncher) Step(v int) {
	for i := 0; i < fl.Threads; i++ {
		fl.Flows[i].Stepper <- v
	}
}

func (fl *FlowLauncher) AutoStep(delay time.Duration, endChan chan *Params) {
	loop := true

	// swallow statuses
	go func() {
		for stat := range fl.CStat {
			fmt.Println("          -------------> Status", stat)
		}
		fmt.Println("loop stoppped")
	}()

	go func() {
		for loop {
			time.Sleep(delay)
			if loop {
				fmt.Println("          (Stepping)")
				go fl.Step(1)
			}
		}
		fmt.Println("stepper loop stoppped")
	}()

	res := <-fl.iEnd
	loop = false

	fmt.Println("signal CEnd")
	if endChan != nil {
		endChan <- res
	}
}

// p are initial environment properties
// run this workflow with this number of threads and gather the success/fail response statistics
func (fl *FlowLauncher) Exec(p Props) {

	// make fresh chanels on each exec
	fl.CStat = make(chan *Params)
	fl.iEnd = make(chan *Params)

	fmt.Println("workflow launcher", fl.Name, " with", fl.Threads, "threads")
	var waitGroup sync.WaitGroup

	endParams := MakeParams()
	endParams.Props = p

	fl.Flows = make([]*Workflow, fl.Threads, fl.Threads)

	// new stats for this run
	fl.LastRunResult = NewFlowLaunchResult()

	// fire off some threads
	for i := 0; i < fl.Threads; i++ {
		flow := fl.FlowFunc(i)
		fl.Flows[i] = flow

		// only realy need to do this once - but hey
		endParams.FlowName = flow.Name

		fmt.Println("workflow launch", flow.Name, "with threadid", i)

		if flow == nil {
			fmt.Println(flow.Name, "has no flow")
			return
		}

		if flow.Start == nil {
			fmt.Println(flow.Name, "has no flow start")
			return
		}

		if flow.End == nil {
			fmt.Println(flow.Name, "has no flow end")
			return
		}

		// TOOD - a function to vary the params per thread
		params := MakeParams()
		params.Props = p
		params.FlowName = flow.Name
		params.ThreadId = i

		fmt.Println("firing thread", params)

		waitGroup.Add(1)
		go flow.Exec(params)

		// loop round forwarding status updates
		go func() {
			for stat := range flow.C {
				fl.LastRunResult.AddResult(stat)
				fmt.Println("got and pushing stat", stat.ThreadId, stat.TaskName)
				fl.CStat <- stat // tiger any stats change chanel (e.g. for push messages like websockets)
			}
			fmt.Println("launcher status loop stoppped")
		}()

		// set up the flow end trigger
		go func() {
			par := <-flow.End.Trigger()
			fmt.Println("got flow end", par.ThreadId, flow.End.GetName())
			// collect all end event triggers - last par wins
			endParams.Status = endParams.Status + par.Status

			// close the flow status chanel
			close(flow.C)
			waitGroup.Done()
		}()
	}

	go func() {
		waitGroup.Wait()
		fmt.Println("completed launcher", fl.Name, "with", fl.Threads, "threads")
		// trigger end
		fl.LastRunResult.Completed = true
		close(fl.CStat)

		fl.iEnd <- endParams
	}()
}

func (fl FlowLauncher) GetStructure() FlowStruct {
	// make a flow just so we can render it in json
	f := fl.FlowFunc(0)
	return f.GetStructure()
}
