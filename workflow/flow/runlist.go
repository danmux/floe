package flow

import (
	"github.com/golang/glog"
)

// TODO - not used yet - but this would be the historical set of lauch results for a given flow
type Run struct {
	Id     string
	Result *FlowLaunchResult
}

type RunList struct {
	Name string

	Runs []Run // a set of response stats by task id in our workflow for the last run
}

func (rl *RunList) AddRun(result *FlowLaunchResult) {
	glog.Info("blah")
}
