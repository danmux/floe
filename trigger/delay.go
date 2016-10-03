package trigger

import (
	"io"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/par"
)

type DelayTrigger struct {
	delay  time.Duration
	config task.TaskConfig
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
func (ft *DelayTrigger) Exec(ctx *task.Context, p *par.Params, out *io.PipeWriter) {
	log.Info("executing delay ", p.Complete)

	time.Sleep(ft.delay)

	out.Write([]byte("Delay triggered\n"))

	p.Response = "node done"
	p.Status = par.StSuccess
	return
}

func (ft *DelayTrigger) Config() task.TaskConfig {
	return ft.config
}
