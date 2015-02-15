package tasks

import (
	f "floe/workflow/flow"
	"github.com/golang/glog"
	"io"
)

type CustomTask struct {
	customFunc CustomExecFunc
	config     f.TaskConfig
}

type CustomExecFunc func(t *f.TaskNode, p *f.Params, out *io.PipeWriter)

func (ft *CustomTask) Type() string {
	return "custom_task"
}

func MakeCustomTask(customFunc CustomExecFunc) *CustomTask {
	return &CustomTask{
		customFunc: customFunc,
	}
}

// params are passed in and mutated with results
func (ft *CustomTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing custom task ", p.Complete)

	ft.customFunc(t, p, out)

	return
}

func (ft *CustomTask) Config() f.TaskConfig {
	return ft.config
}
