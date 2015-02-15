package flow

import (
	"github.com/golang/glog"
)

// a workflow defines the start and ends and some channels and timers to step through
// the tasks themselves know which other task to call
// a workflow is created and run for each thread of a flow launcher
type Workflow struct {
	Name           string
	Start          *TaskNode                    // start is used for a normal workflow there is only one start and it must be a tasknode
	End            TriggeredTaskNode            // any node to end on
	Params         *Params                      // initial condition parameters
	C              chan *Params                 // the chanel that tasknodes attach to and send status updates
	Stepper        chan int                     // inbound channel - to provide stepping
	TaskNodes      map[string]TriggeredTaskNode // map by name of all our nodes
	Stop           bool                         // set true to stop this threads flow - or to mark it stopped
	IgnoreTriggers bool                         // set by the first trigger in the flow - stops other triggers from firing
}

func MakeWorkflow() *Workflow {
	return &Workflow{
		C:              make(chan *Params),
		Stepper:        make(chan int),
		TaskNodes:      make(map[string]TriggeredTaskNode),
		IgnoreTriggers: false,
	}
}

func (w *Workflow) registerNode(tn TriggeredTaskNode) {

	node, in := w.TaskNodes[tn.Id()]

	if in {
		// fail loud if they are not actually the same node being registered more than once!
		if node != tn {
			panic("two nodes with the same id " + tn.Id())
		}
	}

	w.TaskNodes[tn.Id()] = tn
}

// Make a task node with the given task and register it with the workflow
func (w *Workflow) MakeTaskNode(name string, t Task) *TaskNode {
	tn := &TaskNode{
		id:   MakeID(name),
		name: name,
		C:    make(chan *Params, 1), // a buffer of one - as we always send the end even if no one is listening
	}
	tn.SetTask(t)
	tn.SetWorkFlow(w)
	w.registerNode(tn)
	return tn
}

func (w *Workflow) MakeMergeNode(name string) *MergeNode {
	mn := &MergeNode{
		id:    MakeID(name),
		name:  name,
		first: true,
		C:     make(chan *Params, 1), // a buffer of one - as we always send the end even if no one is listening
	}
	mn.SetWorkFlow(w)
	w.registerNode(mn)
	return mn
}

func (w *Workflow) MakeTriggerNode(name string, t Task) *TaskNode {
	tn := w.MakeTaskNode(name, t)
	tn.tType = "trigger"
	return tn
}

func (w *Workflow) SetStart(tn *TaskNode) {
	tn.SetWorkFlow(w)
	w.registerNode(tn)
	w.Start = tn
}

func (w *Workflow) SetEnd(tn TriggeredTaskNode) {
	w.registerNode(tn)
	w.End = tn
}

func (w *Workflow) Exec(p *Params) {
	w.Params = p
	glog.Info("workflow start ", w.Name)
	if w.Start != nil {
		w.Start.Exec(p)
		return
	}
	glog.Error("missing start node - make sure you called SetStart passing in a node")
}

func (w *Workflow) StartTriggers(p *Params) {
	w.Params = p
	glog.Info("starting workflow triggers ", w.Name)

	for _, n := range w.TaskNodes {
		// launch all none merge nodes
		if n.Type() == "trigger" {
			go n.Exec(p)
		}
	}
}

// a flow structure is a structureal relationship of nodes and edges that can be rendered in json
type FlowStruct struct {
	Id    string
	Name  string
	Order int
	Nodes []Node
	Edges []Edge
}

func (f Workflow) GetStructure(order int) FlowStruct {

	fs := FlowStruct{
		Id:    MakeID(f.Name),
		Name:  f.Name,
		Order: order,
		Nodes: make([]Node, 0, 5),
		Edges: make([]Edge, 0, 5),
	}

	for _, n := range f.TaskNodes {
		glog.Info(n)

		node := Node{
			Id:     MakeID(n.Name()),
			Name:   n.Name(),
			Type:   n.Type(),
			Config: n.Config(),
		}
		fs.Nodes = append(fs.Nodes, node)

		edges := n.Edges()
		glog.Info(edges)
		fs.Edges = append(fs.Edges, edges...)
	}

	return fs
}
