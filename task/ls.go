package task

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

// LsTask returns the file listing in params props
type LsTask struct {
	path   string
	config TaskConfig
}

// Type returns the text description of this task
func (ft *LsTask) Type() string {
	return "list directory"
}

// MakeLsTask makes a new LsTask passing in the path to list
func MakeLsTask(path string) *LsTask {
	return &LsTask{
		path: path,
		config: TaskConfig{
			Command: fmt.Sprintf("ls: %v", path),
		},
	}
}

// Exec runs this task, params p are passed in and mutated with results
func (ft *LsTask) Exec(ctx *Context, p *par.Params, out *io.PipeWriter) {
	log.Info("LsTask.Execute")

	path, ok := p.Props["path"]

	// if no passed in path use default
	if !ok {
		path = ft.path
	}

	if path == "" {
		p.Status = par.StFail
		p.Response = "no path specified"
		return
	}

	// this is mandatory node
	path = filepath.Join(ctx.WorkspacePath, path)

	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		p.Props[fmt.Sprint(f.Name())] = ""
	}

	p.Response = "list directory done"
	p.Status = par.StSuccess

	return
}

func (ft *LsTask) Config() TaskConfig {
	return ft.config
}
