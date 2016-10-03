package floe

import (
	"io"
	"strings"

	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/par"
)

// Edge is a struct for reporting e.g. for json marshalling the graph of nodes
type Edge struct {
	Name string
	From string
	To   string
}

// Node the interface for all nodes in a floe
type Node interface {
	// exec fills in and returns the params
	Exec(p *par.Params)
	Type() string
	Name() string
	ID() string
	Edges() []Edge
	SetStream(*io.PipeWriter)
	Config() task.TaskConfig

	setMergeTrigger()
	fireDoneChan(p *par.Params)
	doneChan() chan *par.Params
	setContext(launchContext)
}

// MakeID makes a file friendly ID from the name. TODO - make html friendly id
func MakeID(name string) string {
	s := strings.Split(strings.ToLower(strings.TrimSpace(name)), " ")
	ns := strings.Join(s, "-")
	s = strings.Split(ns, ".")
	return strings.Join(s, "-")
}
