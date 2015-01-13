package tasks

import (
	f "floe/workflow/flow"
	"github.com/golang/glog"
	"io"
	"time"
)

type DelayTask struct {
	delay time.Duration
}

func (ft *DelayTask) Type() string {
	return "delay"
}

func MakeDelayTask(delay time.Duration) *DelayTask {
	return &DelayTask{
		delay: delay,
	}
}

// params are passed in and mutated with results
func (ft *DelayTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing delay ", p.Complete)

	time.Sleep(ft.delay)

	p.Response = "node done"
	p.Status = f.SUCCESS
	return
}
