package tasks

import (
	f "floe/workflow/flow"
	"io"

	"github.com/golang/glog"
)

type CustomTask struct {
	customFunc CustomExecFunc
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
