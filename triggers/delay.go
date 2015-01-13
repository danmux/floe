package tasks

import (
	f "floe/workflow/flow"
	"github.com/golang/glog"
	"io"
	"time"
)

type DelayTrigger struct {
	delay time.Duration
}

func (ft *DelayTrigger) Type() string {
	return "delay"
}

func MakeDelayTrigger(delay time.Duration) *DelayTrigger {
	return &DelayTrigger{
		delay: delay,
	}
}

// params are passed in and mutated with results
func (ft *DelayTrigger) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing delay ", p.Complete)

	time.Sleep(ft.delay)

	p.Response = "node done"
	p.Status = f.SUCCESS
	return
}
