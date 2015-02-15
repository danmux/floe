package tasks

import (
	f "floe/workflow/flow"
	"fmt"
	"github.com/golang/glog"
	"io"
	"time"
)

type DelayTask struct {
	delay  time.Duration
	config f.TaskConfig
}

func (ft *DelayTask) Type() string {
	return "delay"
}

func MakeDelayTask(delay time.Duration) *DelayTask {
	return &DelayTask{
		delay: delay,
		config: f.TaskConfig{
			Command: fmt.Sprintf("delay: %v", delay),
		},
	}
}

// params are passed in and mutated with results
func (ft *DelayTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing delay ", p.Complete)

	time.Sleep(ft.delay)

	out.Write([]byte("Delay complete\n"))

	p.Response = "node done"
	p.Status = f.SUCCESS
	return
}

func (ft *DelayTask) Config() f.TaskConfig {
	return ft.config
}
