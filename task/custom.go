package task

import (
	"io"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

type CustomTask struct {
	task   CustomExecTask
	config TaskConfig
}

// CustomExecTask is what we need to implement to code up any custom work
type CustomExecTask interface {
	ExecFunc(ctx *Context, p *par.Params, out *io.PipeWriter)
	Description() string
}

func (ct *CustomTask) Type() string {
	return "custom_task"
}

func MakeCustomTask(custom CustomExecTask) *CustomTask {
	return &CustomTask{
		task: custom,
		config: TaskConfig{
			Command: custom.Description(),
		},
	}
}

// params are passed in and mutated with results
func (ct *CustomTask) Exec(ctx *Context, p *par.Params, out *io.PipeWriter) {
	log.Info("executing custom task ", p.Complete)

	ct.task.ExecFunc(ctx, p, out)

	return
}

func (ct *CustomTask) Config() TaskConfig {
	return ct.config
}
