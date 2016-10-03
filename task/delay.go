package task

import (
	"fmt"
	"io"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

type DelayTask struct {
	delay  time.Duration
	config TaskConfig
}

func (ft *DelayTask) Type() string {
	return "delay"
}

func MakeDelayTask(delay time.Duration) *DelayTask {
	return &DelayTask{
		delay: delay,
		config: TaskConfig{
			Command: fmt.Sprintf("delay: %v", delay),
		},
	}
}

// params are passed in and mutated with results
func (ft *DelayTask) Exec(ctx *Context, p *par.Params, out *io.PipeWriter) {
	log.Info("DelayTask.Exec", ft.delay)

	time.Sleep(ft.delay)

	out.Write([]byte("Delay complete\n"))

	p.Response = "node done"
	p.Status = par.StSuccess
	return
}

func (ft *DelayTask) Config() TaskConfig {
	return ft.config
}
