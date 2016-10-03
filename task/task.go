package task

import (
	"io"

	"github.com/floeit/floe/workfloe/par"
)

// Task is the interface that the task nodes hod that actually do the work.
// These task types are added to floe/task
type Task interface {
	// exec fills in and returns the params
	Exec(ctx *Context, p *par.Params, out *io.PipeWriter)
	Type() string
	Config() TaskConfig // json representation of the config for the node
}

// TaskConfig holds any useful public information about a specific task
// at the moment this is just used for display
type TaskConfig struct {
	Command string
}

// Context holds some context for the task to operate in
type Context struct {
	WorkspacePath   string
	TriggerDataPath string
}
