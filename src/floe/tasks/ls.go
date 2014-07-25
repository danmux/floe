package tasks

import (
	f "floe/workflow/flow"
	"fmt"
	"io/ioutil"
)

type LsTask struct {
	path string
}

func (ft *LsTask) Type() string {
	return "list directory"
}

func MakeLsTask(path string) *LsTask {
	return &LsTask{
		path: path,
	}
}

// params are passed in and mutated with results
func (ft *LsTask) Exec(t *f.TaskNode, p *f.Params) {
	fmt.Println("executing list directory")

	path, ok := p.Props["path"]

	// if no passed in path use defualt
	if !ok {
		path = ft.path
	}

	if path == "" {
		p.Status = f.FAIL
		p.Response = "no path specified"
		return
	}

	// this is mandatory node
	path = t.Flow.Params.Props["workspace"] + "/" + path

	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		p.Props[fmt.Sprint(f.Name())] = ""
	}

	p.Response = "list directory done"
	p.Status = f.SUCCESS

	return
}
