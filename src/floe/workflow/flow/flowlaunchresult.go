package flow

import (
	"time"
)

// TODO - keep each individual thread result?? - too much for 1M threads for example, but could consider it for under 10 threads
type FlowLauncherStats struct {
	Complete int
	Failed   int
}

type FlowLaunchResult struct {
	FlowId    string
	Start     time.Time
	Duration  time.Duration
	Completed bool
	Results   map[string]FlowLauncherStats // a set of response stats by task id in our workflow for the last run
}

func NewFlowLaunchResult() *FlowLaunchResult {
	flr := &FlowLaunchResult{
		Start:     time.Now(),
		Completed: false,
		Results:   make(map[string]FlowLauncherStats),
	}
	return flr
}

func (f *FlowLaunchResult) AddResult(p *Params) {
	stat, ok := f.Results[p.TaskId]
	if !ok {
		stat = FlowLauncherStats{}
	}

	stat.Complete = stat.Complete + 1
	if p.Status >= 300 {
		stat.Failed = stat.Failed + 1
	}

	f.Results[p.TaskId] = stat
}
