package flow

import (
	"fmt"
	"sync"
)

type GetFlowFunc func(threadId int) *Workflow

type FlowLauncher struct {
	Name     string
	FlowFunc GetFlowFunc
	Threads  int
	Flows    []*Workflow
	Props    *Props
	CStat    chan *Params
	CEnd     chan *Params
}

// the launchers trigger is the end tasknodes trigger
func (fl *FlowLauncher) Trigger() chan *Params {
	return fl.CEnd
}

func (fl *FlowLauncher) Step(v int) {
	for i := 0; i < fl.Threads; i++ {
		fl.Flows[i].Stepper <- v
	}
}

func MakeFlowLauncher(name string, flowFunc GetFlowFunc, threads int) *FlowLauncher {

	return &FlowLauncher{
		Name:     name,
		FlowFunc: flowFunc,
		Threads:  threads,
		CStat:    make(chan *Params),
		CEnd:     make(chan *Params),
	}
}

// p are initial environment properties
func (fl *FlowLauncher) Exec(p Props) {

	fmt.Println("workflow launcher", fl.Name, " with", fl.Threads, "threads")
	var waitGroup sync.WaitGroup

	endParams := MakeParams()
	endParams.Props = p

	fl.Flows = make([]*Workflow, fl.Threads, fl.Threads)

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
		loop := true
		go func() {
			for loop {
				stat := <-flow.C // each tasknode sends status to the flows main channel
				fl.CStat <- stat
			}
			fmt.Println("launcher loop stoppped")
		}()

		// set up the flow end trigger
		go func() {
			par := <-flow.End.Trigger()
			fmt.Println("got flow end", par.ThreadId)
			// collect all end event triggers - last par wins
			endParams.Status = endParams.Status + par.Status
			loop = false
			waitGroup.Done()
		}()
	}

	go func() {
		waitGroup.Wait()
		fmt.Println("completed launcher", fl.Name, "with", fl.Threads, "threads")
		// trigger end
		fl.CEnd <- endParams
	}()
}

func (fl FlowLauncher) GetStructure() FlowStruct {
	// make a flow just so we can render it in json
	f := fl.FlowFunc(0)
	return f.GetStructure()
}
