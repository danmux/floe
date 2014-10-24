package flow

import (
	"third_party/github.com/golang/glog"
)

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
