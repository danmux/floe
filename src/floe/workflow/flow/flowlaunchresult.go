package flow

import (
	"bufio"
	"errors"
	"io"
	"third_party/github.com/golang/glog"
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

type FlowLaunchResult struct {
	Error        string
	FlowId       string
	Start        time.Time
	Duration     time.Duration
	Completed    bool
	Results      map[string]*FlowLauncherStats // a set of response stats by task id in our workflow for the last run
	TotalThreads int
}

func NewFlowLaunchResult(threads int) *FlowLaunchResult {
	flr := &FlowLaunchResult{
		Start:        time.Now(),
		Completed:    false,
		Results:      make(map[string]*FlowLauncherStats),
		TotalThreads: threads, // how many threads should be executed
	}
	return flr
}

func (f *FlowLaunchResult) AddTask(taskId string) (*FlowLauncherStats, error) {
	stat, ok := f.Results[taskId]
	if !ok {
		rp, wp := io.Pipe()
		stat = &FlowLauncherStats{
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
		}(stat, rp)

		f.Results[taskId] = stat
		return stat, nil
	} else {
		return nil, errors.New("attempting to add none unique task name " + taskId)
	}
}

func (f *FlowLaunchResult) AddStatusOrResult(id string, complete bool, status int) {
	stat, ok := f.Results[id]
	if !ok {
		panic("a task id was changed or added after initialisation of the flow")
	}
	// mark it at least one percent complete so we can see that it is in progress
	stat.PercentComplete = 1

	if complete {
		stat.Complete = stat.Complete + 1
		if status > 0 {
			stat.Failed = stat.Failed + 1
		}

		stat.PercentComplete = (stat.Complete * 100) / f.TotalThreads
	}

	f.Results[id] = stat
}
