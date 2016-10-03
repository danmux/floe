package floe

import (
	"os"
	"sync"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/hist"
	"github.com/floeit/floe/workfloe/par"
	"github.com/floeit/floe/workfloe/space"
)

// LaunchContext is blah
type launchContext interface {
	statusChan() chan *par.Params
	isRunning() bool
	space() *task.Context
}

// Launcher services one or multiple Workfloe threads. Each Workfloe is constructed by the WorkfloeFunc in floeFunc.
// These can be used to fire off concurrent test Workfloes for example as a load test.
// Each FloeLauncher is reused for each execution of N Workfloes.
// Each time a run is executed on a launcher a new lastRunResult is created which will capture metrics from all threads.
// Some specific details such as the command output will only be captured from the first thread.
type Launcher struct {
	mu            sync.Mutex         // mutex to protect some stuff
	name          string             // the name of the launcher - acquired from the launchable
	id            string             // the id of the launcher - acquired from the launchable
	order         int                // what order to show this in in its parent project
	threads       int                // the number of threads to launch when this launcher is started
	initProps     *par.Props         // an initial set of properties set from the launchable
	paramChan     chan *par.Params   // the channel to listen for params from node to node as they are executed
	conf          *space.Conf        // any static config as set on the floe - or accessible on the project
	initial       *Launcher          // a flow to always run before this one
	floes         []*Workfloe        // each active thread creates a full workfloe in memory - so the implementor of tasks does not have to worry about thread conflicts
	runList       *hist.RunList      // the history of runs for this launcher
	running       bool               // if this launcher has active threads
	floeFunc      WorkfloeFunc       // the func that will return a constructed Workfloe
	iEnd          chan *par.Params   // internal end chanel for auto stepper
	endParams     *par.Params        // the set of params this floe ended with
	lastRunResult *hist.LaunchResult // a set of response stats by task id in our workfloe for the current or last executed run

	// trigger       *Launcher          // a floe containing any triggers for this floe
	sampleFloe *Workfloe      // a single instantiated floe so we can see what the floe is
	obs        StatusObserver // a status observer
}

// NewWorkfloe adds a workfloe to the launcher with the given threadID
func (fl *Launcher) NewWorkfloe(threadID int) *Workfloe {
	w := fl.floeFunc(threadID)
	w.id = fl.id
	w.setContext(fl)
	return w
}

// Start is main entry point this may launch a dependant initial workfloe - and block on that
func (fl *Launcher) Start(delay time.Duration, endChan chan *par.Params) {
	// wipe previous results
	fl.trashLastResults()

	if fl.initial != nil { // do we have another floe that must be run before this one

		ec := make(chan *par.Params)

		go fl.initial.Start(delay, ec)

		// block waiting for initial to complete
		res := <-ec

		log.Info("initial floe end result ", res)

		if res.Status == 0 {
			log.Info("INIT FLOW SUCCEEDED")
		} else {
			log.Info("INIT FLOW FAILED")
			return
		}
	}

	log.Info("starting threads ", fl.id)

	allLaunched := make(chan bool)
	go fl.exec(allLaunched)
	<-allLaunched

	log.Info("launched", fl.id)

	if fl.lastRunResult == nil {
		panic("nil lastRunResult") // TODO - still needed for debug?
	}

	log.Info("starting stepper ", fl.id)

	go fl.autoStep(delay, endChan)
}

// Step triggers all the threads stepper - to pace the execution of each thread
func (fl *Launcher) Step(v int) {
	// cant step if there was a problem and we didn't make all threads
	if len(fl.floes) < fl.threads {
		log.Errorf("not enough threads %s %d < %d", fl.id, len(fl.floes), fl.threads)
		return
	}
	log.Debugf("  -- stepping %d threads", len(fl.floes))
	// lock them all whilst firing stepper messages
	fl.mu.Lock()
	for i := 0; i < fl.threads; i++ {
		select {
		case fl.floes[i].stepper <- v: // only send the step if they are waiting
		default:
		}
	}
	fl.mu.Unlock()
}

// newLauncher makes a new launcher
func newLauncher(launchable Launchable, threads int) *Launcher {
	fl := &Launcher{
		initProps: launchable.GetProps(),
		name:      launchable.Name(),
		id:        launchable.ID(),
		floeFunc:  launchable.FloeFunc,
		threads:   threads,
	}
	fl.sampleFloe = fl.NewWorkfloe(0) // make the sample floe structure by calling the floe function
	return fl
}

// StartTrigger is the main entry point to a launcher wrapping a
func (fl *Launcher) startTrigger(delay time.Duration, endChan chan *par.Params) {
	if fl.sampleFloe == nil {
		panic("need a sample floe to start the trigger")
	}

	fl.trashLastResults()
	go fl.execTrigger()

	go fl.autoStep(delay, endChan)
}

func (fl *Launcher) statusChan() chan *par.Params {
	return fl.paramChan
}

func (fl *Launcher) space() *task.Context {
	return fl.conf.Context()
}

func (fl *Launcher) isRunning() bool {
	fl.mu.Lock()
	r := fl.running
	fl.mu.Unlock()
	return r
}

func (fl *Launcher) setRunning(r bool) {
	fl.mu.Lock()
	fl.running = r
	fl.mu.Unlock()
}

func (fl *Launcher) pushToObs(thing string, prog interface{}) {
	fl.mu.Lock()
	ob := fl.obs
	fl.mu.Unlock()
	if ob != nil {
		ob.Write("prog", prog)
	}
}

// autoStep allows us to set the pace at which each step can move on
func (fl *Launcher) autoStep(delay time.Duration, endChan chan *par.Params) {
	loop := true
	mu := &sync.Mutex{}

	go func() {
		for {
			mu.Lock()
			l := loop
			mu.Unlock()
			if !l {
				break
			}

			log.Debug("firing step event >>>")

			go fl.Step(1)
			time.Sleep(delay)
		}
		log.Info("stepper loop stopped")
	}()

	log.Info("waiting for launcher to end")
	res := <-fl.iEnd

	mu.Lock()
	loop = false
	mu.Unlock()

	log.Info("endChan <<<")
	if endChan != nil {
		endChan <- res
	}
}

func (fl *Launcher) trashLastResults() {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.lastRunResult = nil
}

func (fl *Launcher) changeHistory() error {
	if fl.runList == nil {
		fl.runList = &hist.RunList{
			FloeID: fl.id,
		}
	}
	// add to the run list if we don't have it already and save it
	fl.runList.AddRun(fl.lastRunResult)
	return fl.runList.Save(fl.conf.HistoryStore)
}

// EnforceTidyDeskPolicy clear out the workspace if required - backs up last workspace
func (fl *Launcher) EnforceTidyDeskPolicy(p par.Props) bool {
	ws := fl.conf.WorkspacePath()
	if p[par.KeyTidyDesk] != "keep" {
		log.Info("removing and moving ", ws)
		err := os.RemoveAll(ws + "_old")
		if err != nil {
			if !os.IsNotExist(err) {
				log.Error(err)
				fl.lastRunResult.Error = err.Error()
				return false
			}
		}
		err = os.Rename(ws, ws+"_old")
		if err != nil {
			if !os.IsNotExist(err) {
				log.Error(err)
				fl.lastRunResult.Error = err.Error()
				return false
			}
		}
		log.Info("removing and moving done")
	}
	// make sure we have a workspace
	if err := os.MkdirAll(ws, 0777); err != nil {
		log.Warning(err)
		// expect an existing folder if its set to keep
		if p[par.KeyTidyDesk] == "keep" {
			return true
		}
		log.Error(err)
		fl.lastRunResult.Error = err.Error()
		return false
	}
	// now make sure we have the triggers state folder in place
	ts := fl.conf.TriggerDataPath()
	if err := os.MkdirAll(ts, 0777); err != nil {
		log.Error(err)
		fl.lastRunResult.Error = err.Error()
		return false
	}
	// and the history state folder in place
	ts = fl.conf.HistoryDataPath()
	if err := os.MkdirAll(ts, 0777); err != nil {
		log.Error(err)
		fl.lastRunResult.Error = err.Error()
		return false
	}
	log.Info("all folders in place")
	return true
}

// exec launches all threads for this launcher and waits for all threads to finish in a goroutine
func (fl *Launcher) exec(allLaunched chan bool) {
	if !fl.prep(false) {
		return
	}

	fl.setRunning(true)

	log.Info("workfloe launcher ", fl.id, " with ", fl.threads, " threads")

	var (
		waitComplete sync.WaitGroup // to wait for all threads to complete
		waitLaunched sync.WaitGroup // to wait for all threads to have been started
	)

	// fire off some threads concurrently
	for i := 0; i < fl.threads; i++ {
		waitComplete.Add(1)
		waitLaunched.Add(1)

		go fl.execOneFloe(i, &waitComplete, &waitLaunched, false)
	}

	// wait for all threads to be launched
	waitLaunched.Wait()

	// and tell our caller that we are launched
	allLaunched <- true

	// now lets wait for all the threads to finish
	// TODO a timeout as well in case of bad floes...
	go func() {
		waitComplete.Wait()
		// once we get past the waitgroup then all threads have completed
		log.Info("completed launcher ", fl.id, " with ", fl.threads, " threads")
		// mark status
		fl.lastRunResult.Completed = true

		// close the status channel
		fl.mu.Lock()
		close(fl.paramChan)
		fl.mu.Unlock()

		log.Info("trigger end")
		// and trigger the end event
		fl.iEnd <- fl.endParams
		log.Info("end triggered")

		fl.setRunning(false)

		// update run history now its finished
		fl.changeHistory()
	}()
}

// execTrigger is the main execution method for a trigger floe - it synchronously starts the floe
func (fl *Launcher) execTrigger() {
	if !fl.prep(true) {
		return
	}

	log.Info("workfloe trigger ", fl.name)

	fl.execOneFloe(0, nil, nil, true)

	log.Info("completed trigger ", fl.name)

	// mark status
	fl.lastRunResult.Completed = true

	// close the status channel
	fl.mu.Lock()
	close(fl.paramChan)
	fl.mu.Unlock()

	// and trigger the end event
	fl.iEnd <- fl.endParams
}

// prep sets up all the things prior to an execution about to begin. Returns true if all preparations are good
func (fl *Launcher) prep(isTrigger bool) bool {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// don't let a launcher launch another batch if it is still running another one
	if fl.running {
		log.Warning(fl.id, "can not start an already running launcher")
		return false
	}
	// the final set of params set at the end of the floe - last finishing thread wins
	fl.endParams = &par.Params{
		Props: *fl.initProps, // set up with initial props
	}
	// check some conditions...
	log.Info("workfloe prep", fl.id)
	floe := fl.sampleFloe
	if floe == nil {
		log.Error(fl.id, "has no floe")
		return false
	}
	if !isTrigger && floe.start == nil {
		log.Error(fl.id, "has no floe start")
		return false
	}
	if floe.end == nil {
		log.Error(fl.id, "has no floe end")
		return false
	}

	// make fresh channels on each exec as they were probably closed
	fl.paramChan = make(chan *par.Params, 10) // can hold some messages - to avoid some deadlock TODO understand if this is needed
	fl.iEnd = make(chan *par.Params, 10)

	if fl.EnforceTidyDeskPolicy(*fl.initProps) == false {
		fl.endParams.Status = par.StFail
		fl.iEnd <- fl.endParams
		return false
	}

	log.Info(fl.id, "creating threads ", fl.threads)
	fl.floes = make([]*Workfloe, fl.threads, fl.threads)

	return true
}

// execOneFloe launch one floe thread if isTrigger is set then the floe is launched in the specific trigger style
func (fl *Launcher) execOneFloe(i int, waitComplete *sync.WaitGroup, waitLaunched *sync.WaitGroup, isTrigger bool) {

	// create a new workfloe
	floe := fl.NewWorkfloe(i)

	// for the first thread make the results object including
	if i == 0 {
		// NewLaunchResults make the object to capture the result of the current run
		// and set the result stream objects on the nodes in this floe. It also starts ranging CStat channel
		// capturing any node parameter updates.
		fl.lastRunResult = hist.NewLaunchResult(fl.threads, fl.pushToObs, fl.paramChan)

		// for each of our task nodes add a task to the results
		for _, n := range floe.taskNodes {
			log.Infof("add <%s> to run result: ", n.ID())
			cons, err := fl.lastRunResult.AddNode(n.ID(), n.Name())
			if err != nil {
				log.Error("dodgy nodes ", err)
			} else {
				n.SetStream(cons.CommandStream)
			}
		}

		// add lastRunResult to our history
		if err := fl.changeHistory(); err != nil {
			log.Errorf("%s could not save run history: %s", fl.id, err.Error())
			return
		}
	}

	// save it for later
	fl.floes[i] = floe

	log.Info("workfloe launch ", fl.id, " with thread id ", i)

	// TODO - here you would inject any function to vary the params per thread
	// copy the params and add initial props
	startPar := &par.Params{
		Props: *fl.initProps,
	}

	log.Info("firing floe with params ", startPar)

	// and fire of the workfloe
	if isTrigger {
		go floe.execTriggers(startPar)
	} else {
		go floe.exec(startPar)
	}

	// at this point the thread is in play
	if waitLaunched != nil {
		waitLaunched.Done()
	}

	// wait for the floe end trigger - the end registered node will fire this
	// TODO consider a timeout on waiting for this End node to be done
	endPar := <-floe.end.doneChan()

	log.Infof("got floe end %d %s", i, floe.end.Name())
	// collect all end event triggers - last par wins
	fl.endParams.Status = fl.endParams.Status + endPar.Status

	// update duration in seconds
	fl.lastRunResult.Duration = int(time.Since(fl.lastRunResult.Start) / time.Second)

	if endPar.Status == par.StFail {
		fl.lastRunResult.Error = endPar.Response
		if fl.lastRunResult.Error == "" {
			fl.lastRunResult.Error = "run failed"
		}
	}

	if waitComplete != nil {
		waitComplete.Done()
	}
}

// this stops all nodes from sending status updates and stops nodes from triggering exec on any next nodes
func (fl *Launcher) exterminateExterminate() {
	// if this launcher is not running then we have nothing to do
	if !fl.isRunning() {
		return
	}
	// first mark the launcher as not running
	fl.setRunning(false)
	// then close the status chanel to stop the lastResult ranger
	close(fl.paramChan)
}

// getStructure return the floe structure - for interfaces
func (fl *Launcher) getStructure() Structure {
	// make a floe just so we can render it in json
	s := fl.sampleFloe.structure(fl.order)
	s.ID = fl.id
	s.Name = fl.name
	return s
}
