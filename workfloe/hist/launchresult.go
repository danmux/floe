package hist

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/buildkite/terminal"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

// the stats for multiple threads per task
type nodeMetrics struct {
	Complete        int
	Failed          int
	PercentComplete int
}

type Console struct {
	CommandOutput  []string       // lines of shell output
	RenderedOutput string         // html'd CommandOutput
	CommandStream  *io.PipeWriter // the writer that is used to pipe stdout and stdErr - and captured in CommandOutput
}

// stepResult captures the outcome of a particular task in the floe
type nodeResult struct {
	// Name The display name
	Name string
	// Stats are the set of metrics for all executions of this node all launcher workfloe threads will be counted in here
	Metrics *nodeMetrics
	// Console captures the nodes executing task only the first Workfloe will be captured
	Console *Console
	// StartParam captures starting Params of the first thread for a given node
	StartParam *par.Params
	// EndParam captures ending Params of the first thread for a given node
	EndParam *par.Params
}

type obsFn func(thing string, prog interface{}) // the observer function signature

// LaunchResult will be created for one invocation of a launch. All outcomes of all workfloe threads will be captured and totalled up.
// Only the console output of the first thread will be captured for each nodes nodeResult
type LaunchResult struct {
	FloeID       string                 // what launcher this belonged to
	RunID        int                    // the run id as assigned by the history
	Reason       string                 // TODO what triggered this run
	Error        string                 // did it finish with any error
	Start        time.Time              // what time it started UTC
	Duration     int                    // duration in seconds
	Completed    bool                   // did it definitely stop
	Results      map[string]*nodeResult // a set of response stats by task id in our workfloe for the last run
	TotalThreads int
	observeFn    obsFn
}

// NewLaunchResult creates a new launchResult and collects all parameter updates from all threads and updates the passed in observer function
func NewLaunchResult(threads int, o obsFn, p chan *par.Params) *LaunchResult {
	log.Info("new LaunchResult result ")
	flr := &LaunchResult{
		Start:        time.Now().UTC(),
		Completed:    false,
		Results:      make(map[string]*nodeResult),
		TotalThreads: threads, // how many threads should be executed
		observeFn:    o,
	}
	// start servicing any updates
	flr.rangeParams(p)
	return flr
}

func (f *LaunchResult) rangeParams(p chan *par.Params) {
	// loop round forwarding status updates - which occur on every transition from one task to the next
	go func() {
		log.Info("waiting on chanel status channel")
		for stat := range p {
			log.Debugf("got status %#v", stat) // TODO less verbose
			f.addStatusOrResult(stat)
		}
		log.Infof("launcher status loop stopped last task: <%s>", f.FloeID)
	}()
}

// AddNode adds a task to the result and returns the console object that captures the task output
func (f *LaunchResult) AddNode(id, name string) (*Console, error) {
	res, found := f.Results[id]
	if found {
		return nil, errors.New("attempting to add none unique task name " + id)
	}
	res = &nodeResult{
		Name: name,
	}
	rp, wp := io.Pipe()
	res.Metrics = &nodeMetrics{}

	res.Console = &Console{
		CommandStream: wp,
		CommandOutput: []string{},
	}
	// start the goroutine to keep pushing the
	go func(s *Console, r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			t := scanner.Text()
			log.Debug(">> console %s \n", t)
			s.CommandOutput = append(s.CommandOutput, t)
		}
		all := strings.Join(s.CommandOutput, "\n")
		log.Debug("===== ALL Commands:")
		log.Debug(all)
		s.RenderedOutput = string(terminal.Render([]byte(all)))
		if err := scanner.Err(); err != nil {
			log.Error("There was an error with the scanner in attached container ", err)
		}
		// push the status change
		f.observeFn("prog", res)
	}(res.Console, rp)

	// and add it to the results
	f.Results[id] = res
	return res.Console, nil
}

func (f *LaunchResult) addStatusOrResult(statusParams *par.Params) {

	id := statusParams.TaskID
	complete := statusParams.Complete
	status := statusParams.Status

	res, ok := f.Results[id]
	if !ok {
		panic("a task id was changed or added after initialisation of the floe " + id)
	}
	metrics := res.Metrics
	// mark it at least one percent complete so we can see that it is in progress
	metrics.PercentComplete = 1

	if complete {
		metrics.Complete = metrics.Complete + 1
		if status > 0 {
			metrics.Failed = metrics.Failed + 1
		}
		metrics.PercentComplete = (metrics.Complete * 100) / f.TotalThreads
		res.EndParam = statusParams
		log.Debug("setting the end params")
	} else {
		if res.StartParam == nil {
			res.StartParam = statusParams
		}
	}

	res.Metrics = metrics
	f.observeFn("prog", res)
}
