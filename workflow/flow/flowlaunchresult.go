package flow

import (
	"bufio"
	"errors"
	"github.com/golang/glog"
	"io"
	"time"
)

// TODO - keep each individual thread result?? - too much for 1M threads for example, but could consider it for under 10 threads
type FlowLauncherStats struct {
	Complete        int
	Failed          int
	PercentComplete int
	CommandOutput   []string       // lines of shell output
	CommandStream   *io.PipeWriter // the writer that is used to pipe stdout and stdErr - and captured in CommandOutput
	reader          *io.PipeReader // read from this to fill the CommandOutput
}

type StepResult struct {
	Stats      *FlowLauncherStats // a set of response stats by task id in our workflow for the last run
	StartParam *Params
	EndParam   *Params
}

type FlowLaunchResult struct {
	Error        string
	FlowId       string
	Start        time.Time
	Duration     time.Duration
	Completed    bool
	Results      map[string]*StepResult // a set of response stats by task id in our workflow for the last run
	TotalThreads int
}

func NewFlowLaunchResult(threads int) *FlowLaunchResult {
	glog.Info("new LaunchResult result ")
	flr := &FlowLaunchResult{
		Start:        time.Now(),
		Completed:    false,
		Results:      make(map[string]*StepResult),
		TotalThreads: threads, // how many threads should be executed
	}
	return flr
}

func (f *FlowLaunchResult) AddTask(taskId string) (*FlowLauncherStats, error) {
	glog.Infof("add result task %s \n", taskId)

	res, ok := f.Results[taskId]
	if !ok {

		res = &StepResult{}

		rp, wp := io.Pipe()
		res.Stats = &FlowLauncherStats{
			CommandStream: wp,
			reader:        rp,
			CommandOutput: []string{},
		}

		// start the threads to monitor the reader
		go func(s *FlowLauncherStats, r io.Reader) {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				t := scanner.Text()
				glog.Infof("%s%s \n", ">> console: ", t)
				s.CommandOutput = append(s.CommandOutput, t)
			}
			if err := scanner.Err(); err != nil {
				glog.Error("There was an error with the scanner in attached container ", err)
			}
		}(res.Stats, rp)

		// and add it to the results
		f.Results[taskId] = res
		return res.Stats, nil
	} else {
		return nil, errors.New("attempting to add none unique task name " + taskId)
	}
}

func (f *FlowLaunchResult) AddStatusOrResult(statusParams *Params) {

	id := statusParams.TaskId
	complete := statusParams.Complete
	status := statusParams.Status

	res, ok := f.Results[id]
	if !ok {
		panic("a task id was changed or added after initialisation of the flow")
	}

	stat := res.Stats
	// mark it at least one percent complete so we can see that it is in progress
	stat.PercentComplete = 1

	if complete {
		stat.Complete = stat.Complete + 1

		if status == 1 {
			stat.Failed = stat.Failed + 1
		}

		stat.PercentComplete = (stat.Complete * 100) / f.TotalThreads

		// if we are in a loop
		if stat.Complete > f.TotalThreads {
			stat.PercentComplete /= stat.Complete
		}

		res.EndParam = statusParams
		glog.Info("setting the endparams <<<<<<<<<<<<")

	} else {
		if res.StartParam == nil {
			res.StartParam = statusParams
		}
	}

	res.Stats = stat
}
