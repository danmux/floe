package flow

import (
	"os"
	"sync"
	"third_party/github.com/golang/glog"
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
// flowlaunchers are persistant in the list of Flows held in the project
// flowlauncher creates N Flows so that each workflow runs in its own thread
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
	Error         string
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
	// cant step if there was a problem and we didnt make all threads
	if len(fl.Flows) < fl.Threads {
		glog.Error("not enough threads")
		return
	}
	glog.Info("<<<<<<<<<<<<<<<<<<<<<<<<<<< step")
	for i := 0; i < fl.Threads; i++ {
		fl.Flows[i].Stepper <- v
	}
}

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

func (fl *FlowLauncher) MakeLaunchResults(tf *Workflow) {

	// new stats for this run
	fl.LastRunResult = NewFlowLaunchResult(fl.Threads)

	// TaskNodes
	for _, n := range tf.TaskNodes {
		glog.Info("add name node ", n.GetName())
		ls, err := fl.LastRunResult.AddTask(MakeID(n.GetName()))
		if err != nil {
			glog.Error("dodgy nodes ", err)
		} else {
			n.SetStream(ls.CommandStream)
		}
	}
}

func (fl *FlowLauncher) TidyDeskPolicy(p Props) bool {
	ws := p[KEY_WORKSPACE]
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
	err = os.Mkdir(ws, 0777)
	if err != nil {
		glog.Error(err)
		fl.Error = err.Error()
		fl.LastRunResult.Error = fl.Error
		return false
	}
	// time.Sleep(5 * time.Second)

	glog.Info("removing and moving done")
	return true
}

// p are initial environment properties
// run a workflow with this number of threads and gather the success/fail response statistics
func (fl *FlowLauncher) Exec(p Props) {

	endParams := MakeParams()
	endParams.Props = p

	// make fresh chanels on each exec as they were probably closed
	fl.CStat = make(chan *Params)
	fl.iEnd = make(chan *Params)

	if fl.LastRunResult == nil {
		fl.LastRunResult = &FlowLaunchResult{}
	}
	fl.Error = ""

	if fl.TidyDeskPolicy(p) == false {
		endParams.Status = FAIL
		fl.iEnd <- endParams
		return
	}

	glog.Info("workflow launcher ", fl.Name, " with ", fl.Threads, " threads")
	var waitGroup sync.WaitGroup

	fl.Flows = make([]*Workflow, fl.Threads, fl.Threads)

	// fire off some threads
	for i := 0; i < fl.Threads; i++ {

		// create a new workflow
		flow := fl.FlowFunc(i)

		// save it for later
		fl.Flows[i] = flow

		// only realy need to do this once - but hey
		endParams.FlowName = flow.Name

		glog.Info("workflow launch ", flow.Name, " with threadid ", i)

		// check some conditions...
		if flow == nil {
			glog.Error(flow.Name, " has no flow")
			return
		}

		if flow.Start == nil {
			glog.Error(flow.Name, " has no flow start")
			return
		}

		if flow.End == nil {
			glog.Error(flow.Name, " has no flow end")
			return
		}

		// for the first thread make the results including capturing any streamed io from the tasks
		if i == 0 {
			fl.MakeLaunchResults(flow)
		}

		// TOOD - here you would inject any function to vary the params per thread
		params := MakeParams()
		params.Props = p
		params.FlowName = flow.Name
		params.ThreadId = i

		glog.Info("firing task with params ", params)

		// build up the thread wait group
		waitGroup.Add(1)

		// and fire of the workflow
		go flow.Exec(params)

		// loop round forwarding status updates
		go func() {
			glog.Info("waiting on chanel flow.C ")
			for stat := range flow.C {
				glog.Info("got status ", stat)
				fl.LastRunResult.AddStatusOrResult(stat.TaskId, stat.Complete, stat.Status)
				fl.CStat <- stat // tiger any stats change chanel (e.g. for push messages like websockets)
			}
			glog.Info("launcher status loop stoppped")
		}()

		// set up the flow end trigger - the end registered node will fire this
		go func() {
			par := <-flow.End.Trigger()
			glog.Info("got flow end ", par.ThreadId, " ", flow.End.GetName())
			// collect all end event triggers - last par wins
			endParams.Status = endParams.Status + par.Status

			// update metrics
			now := time.Now()
			fl.LastRunResult.Duration = now.Sub(fl.LastRunResult.Start)

			// close the flow status chanel
			close(flow.C)

			// count down the waitgroup
			waitGroup.Done()
		}()
	}

	// now lets wait for all the threads to finish - probably TODO a timeout as well in case of bad flows...
	go func() {
		waitGroup.Wait()
		glog.Info("completed launcher ", fl.Name, " with ", fl.Threads, " threads")
		// mark status
		fl.LastRunResult.Completed = true

		// close the status channel
		close(fl.CStat)

		// and trigger the end event
		fl.iEnd <- endParams
	}()
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
	f := fl.FlowFunc(0)
	return f.GetStructure()
}
