package flow

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"sync"
	"time"
)

type GetFlowFunc func(threadId int) *Workflow

// the end user returns these
type LaunchableFlow interface {
	FlowFunc(threadId int) *Workflow
	Name() string
	Id() string
	GetProps() *Props
}

type BaseLaunchable struct {
	name string
	id   string
}

func (b *BaseLaunchable) Init(name string) {
	b.name = name
	b.id = MakeID(name)
}

func (b *BaseLaunchable) Name() string {
	return b.name
}

func (b *BaseLaunchable) Id() string {
	return b.id
}

func (b *BaseLaunchable) DefaultProps() *Props {
	props := Props{}
	props[KEY_WORKSPACE] = "workspace/" + b.id
	props["path"] = "/"
	props[KEY_TIDY_DESK] = "reset" // or keep
	return &props
}

// a load of structures to service a multiple workflow threads- which is passed in via the GetFlowFunc which constructs the workflow
// these can be used to fire off parallel test workflows for example - as a load test
// flowlaunchers are persistant in the list of Flows held in the project
// flowlauncher creates N Flows so that each workflow runs in its own thread
type FlowLauncher struct {
	Name          string
	Id            string
	Order         int
	flowFunc      GetFlowFunc
	Threads       int
	Flows         []*Workflow // each thread creates a full workflow in memory - so the implementor of tasks does not have to wory about thread conflicts
	Props         *Props
	CStat         chan *Params
	iEnd          chan *Params // internal end chanel for auto stepper
	endParams     *Params
	LastRunResult *FlowLaunchResult // a set of response stats by task id in our workflow for the last run
	Error         string
	initial       *FlowLauncher
	trigger       *FlowLauncher
	// TODO - historical stats / logs
}

// make a flow launcher and specify any other initital flow to run before this one
// also define an optional trigger flow
// all flow launchers can be triggered by
func MakeFlowLauncher(launchable LaunchableFlow, threads int, initial *FlowLauncher, trigger *FlowLauncher) *FlowLauncher {

	return &FlowLauncher{
		Props:    launchable.GetProps(),
		Name:     launchable.Name(),
		Id:       launchable.Id(),
		flowFunc: launchable.FlowFunc,
		Threads:  threads,
		initial:  initial,
		trigger:  trigger,
	}
}

// the launchers trigger is the end tasknodes trigger
func (fl *FlowLauncher) Trigger() chan *Params {
	return fl.iEnd
}

// can call own step - perhaps via ui
func (fl *FlowLauncher) Step(v int) {
	// cant step if there was a problem and we didnt make all threads
	if len(fl.Flows) < fl.Threads {
		glog.Error("not enough threads", fl.Id, len(fl.Flows), "<", fl.Threads)
		return
	}
	glog.Info("<<<<<<<<<<<<<<<<<<<<<<<<<<< step")
	for i := 0; i < fl.Threads; i++ {
		fl.Flows[i].Stepper <- v
	}
}

// this allows us to set the pace at which each step can move on
func (fl *FlowLauncher) AutoStep(delay time.Duration, endChan chan *Params) {
	loop := true

	// swallow statuses
	go func() {
		for stat := range fl.CStat {
			glog.Info("<<< status event ", stat)
		}
		glog.Info("loop stoppped")
	}()

	go func() {
		for loop {
			if loop {
				glog.V(2).Infoln("firing step evenct >>>")
				go fl.Step(1)
			}
			time.Sleep(delay)
		}
		glog.Info("stepper loop stoppped")
	}()

	res := <-fl.iEnd
	loop = false

	glog.Info("endChan <<<")
	if endChan != nil {
		endChan <- res
	}
}

func (fl *FlowLauncher) TrashLastResults() {
	fl.LastRunResult = nil
}

// make the object to capture the result
// and set the result stream objects on the nodes in this flow
func (fl *FlowLauncher) MakeLaunchResults(tf *Workflow) {

	// new stats for this run
	fl.LastRunResult = NewFlowLaunchResult(fl.Threads)

	// TaskNodes
	for _, n := range tf.TaskNodes {
		glog.Info("add name node to run result", n.Name())
		ls, err := fl.LastRunResult.AddTask(MakeID(n.Name()))
		if err != nil {
			glog.Error("dodgy nodes ", err)
		} else {
			n.SetStream(ls.CommandStream)
		}
	}
}

func (fl *FlowLauncher) TidyDeskPolicy(p Props) bool {
	ws := p[KEY_WORKSPACE]
	if p[KEY_TIDY_DESK] != "keep" {

		glog.Info("removing and moving ", ws)
		err := os.RemoveAll(ws + "_old")
		if err != nil {
			if !os.IsNotExist(err) {
				glog.Error(err)
				fl.Error = err.Error()
				fl.LastRunResult.Error = fl.Error
				return false
			}
		}
		err = os.Rename(ws, ws+"_old")
		if err != nil {
			if !os.IsNotExist(err) {
				glog.Error(err)
				fl.Error = err.Error()
				fl.LastRunResult.Error = fl.Error
				return false
			}
		}
	}

	err := os.MkdirAll(ws, 0777)
	if err != nil {

		fmt.Println(err)
		// expect an existing folder if its set to keep
		if p[KEY_TIDY_DESK] == "keep" {
			return true
		}
		glog.Error(err)
		fl.Error = err.Error()
		fl.LastRunResult.Error = fl.Error
		return false
	}
	// time.Sleep(5 * time.Second)

	glog.Info("removing and moving done")
	return true
}

func (fl *FlowLauncher) MakeFlow(threadId int) *Workflow {
	w := fl.flowFunc(threadId)
	w.Name = fl.Name
	return w
}

func (fl *FlowLauncher) Prep(isTrigger bool) bool {

	// check we have a good workflow
	flow := fl.MakeFlow(0)

	// the final set of params set at the end of the flow - last finishing thread wins
	fl.endParams = MakeParams()
	fl.endParams.Props = *fl.Props // set up with initial props

	// copy the flow name
	fl.endParams.FlowName = flow.Name

	glog.Info("workflow prep ", flow.Name)

	// check some conditions...
	if flow == nil {
		glog.Error(flow.Name, " has no flow")
		return false
	}

	if !isTrigger && flow.Start == nil {
		glog.Error(flow.Name, " has no flow start")
		return false
	}

	if flow.End == nil {
		glog.Error(flow.Name, " has no flow end")
		return false
	}

	// make fresh chanels on each exec as they were probably closed
	fl.CStat = make(chan *Params)
	fl.iEnd = make(chan *Params)

	fl.Error = ""

	if fl.TidyDeskPolicy(*fl.Props) == false {
		fl.endParams.Status = FAIL
		fl.iEnd <- fl.endParams
		return false
	}

	fl.Flows = make([]*Workflow, fl.Threads, fl.Threads)

	return true
}

// p are initial environment properties
// run a workflow with this number of threads and gather the success/fail response statistics
func (fl *FlowLauncher) Exec() {
	if !fl.Prep(false) {
		return
	}

	glog.Info("workflow launcher ", fl.Name, " with ", fl.Threads, " threads")

	var waitGroup sync.WaitGroup

	// fire off some threads - (in parallel)
	for i := 0; i < fl.Threads; i++ {
		waitGroup.Add(1)

		go fl.execOneFlow(i, &waitGroup, false)
	}

	// now lets wait for all the threads to finish
	// TODO a timeout as well in case of bad flows...
	go func() {
		waitGroup.Wait()
		// once we get past the waitgroup then all threads have completed
		glog.Info("completed launcher ", fl.Name, " with ", fl.Threads, " threads")
		// mark status
		fl.LastRunResult.Completed = true

		// close the status channel
		close(fl.CStat)

		// and trigger the end event
		fl.iEnd <- fl.endParams
	}()
}

// run the trigger flow
func (fl *FlowLauncher) ExecTrigger() {
	if !fl.Prep(true) {
		return
	}

	glog.Info("workflow trigger ", fl.Name)

	fl.execOneFlow(0, nil, true)

	glog.Info("completed trigger ", fl.Name)

	// make sure we mark the single flow thread as stopped so no other triggers can do much
	fl.Flows[0].Stop = true

	// mark status
	fl.LastRunResult.Completed = true

	// close the status channel
	close(fl.CStat)

	// and trigger the end event
	fl.iEnd <- fl.endParams
}

// launch one flow thread if isTrigger is set then the Flow is launched in the specific trigger style
func (fl *FlowLauncher) execOneFlow(i int, waitGroup *sync.WaitGroup, isTrigger bool) {

	// create a new workflow
	flow := fl.MakeFlow(i)

	// save it for later
	fl.Flows[i] = flow

	glog.Info("workflow launch ", flow.Name, " with threadid ", i)

	// TOOD - here you would inject any function to vary the params per thread
	// copy the params and add initial props
	params := MakeParams()
	params.Props = *fl.Props
	params.FlowName = flow.Name
	params.ThreadId = i

	glog.Info("firing task with params ", params)

	// for the first thread make the results object including
	// setting the result stream objects on the nodes in this flow
	if i == 0 {
		fl.MakeLaunchResults(flow)
	}

	// and fire of the workflow
	if isTrigger {
		go flow.StartTriggers(params)
	} else {
		go flow.Exec(params)
	}

	// loop round forwarding status updates
	go func() {
		glog.Info("waiting on chanel flow.C ")
		for stat := range flow.C {
			glog.Info("got status ", stat)

			fl.LastRunResult.AddStatusOrResult(stat)
			fl.CStat <- stat // tiger any stats change chanel (e.g. for push messages like websockets)
		}
		glog.Info("launcher status loop stoppped", i, " ", flow.End.Name())
	}()

	// wait for the flow end trigger - the end registered node will fire this
	par := <-flow.End.DoneChan()

	glog.Info("got flow end ", par.ThreadId, " ", flow.End.Name())
	// collect all end event triggers - last par wins
	fl.endParams.Status = fl.endParams.Status + par.Status

	// update metrics
	now := time.Now()
	fl.LastRunResult.Duration = now.Sub(fl.LastRunResult.Start)

	// close the flow status chanel
	close(flow.C)

	if waitGroup != nil {
		waitGroup.Done()
	}
}

// main entry point - this may launch a dependant initial workflow - and block on that
func (fl *FlowLauncher) Start(delay time.Duration, endChan chan *Params) {
	// wipe previous results
	fl.TrashLastResults()

	if fl.initial != nil {

		ec := make(chan *Params)

		go fl.initial.Start(delay, ec)

		// block waiting for initial to complete
		res := <-ec

		fmt.Println("initial flow end result", res)

		if res.Status == 0 {
			fmt.Println("FLOW SUCCEEDED")
		} else {
			fmt.Println("FLOW FAILED")
			return
		}
	}

	go fl.Exec()

	go fl.AutoStep(delay, endChan)
}

func (fl *FlowLauncher) StartTrigger(delay time.Duration, endChan chan *Params) {
	fl.TrashLastResults()
	go fl.ExecTrigger()

	go fl.AutoStep(delay, endChan)
}

func (fl *FlowLauncher) ExterminateExterminate() {
	if len(fl.Flows) == 0 {
		glog.Warning("stop called on none started launcher")
		return
	}

	// set stop on all active threads
	for i := 0; i < fl.Threads; i++ {
		f := fl.Flows[i]
		if f != nil {
			f.Stop = true
		}
	}
}

// return the flow structure - for interfaces
func (fl FlowLauncher) GetStructure() FlowStruct {
	// make a flow just so we can render it in json
	f := fl.MakeFlow(0)
	return f.GetStructure(fl.Order)
}
