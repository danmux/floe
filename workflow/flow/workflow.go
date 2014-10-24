package flow

import (
	"third_party/github.com/golang/glog"
)

// a workflow defines the start and ends and some channels and timers to step through
// the tasks themselves know which other task to call
type Workflow struct {
	Name      string
	Start     *TaskNode
	End       TriggeredTaskNode
	Params    *Params                      // initial condition parameters
	C         chan *Params                 // the chanel that tasknodes attach to and send status updates
	Stepper   chan int                     // inbound channel - to provide stepping
	TaskNodes map[string]TriggeredTaskNode // map by name of all our nodes
	Stop      bool                         // set true to stop this threads flow
}

func MakeWorkflow(name string) *Workflow {
	return &Workflow{
		C:         make(chan *Params),
		Stepper:   make(chan int),
		Name:      name,
		TaskNodes: make(map[string]TriggeredTaskNode),
	}
}

func (w *Workflow) registerNode(tn TriggeredTaskNode) {
	_, ok := w.TaskNodes[tn.GetName()]
	if !ok {
		w.TaskNodes[tn.GetName()] = tn
	}
}

// TODO - check uniqueness in the flow
func (w *Workflow) MakeTaskNode(name string, t Task) *TaskNode {
	tn := &TaskNode{
		Id:       MakeID(name),
		Name:     name,
		C:        make(chan *Params, 1), // a buffer of one - as we always send the end even if no one is listening
		Triggers: false,
		Flow:     w,
	}

	tn.SetTask(t)

	w.registerNode(tn)

	return tn
}

func (w *Workflow) SetStart(tn *TaskNode) {
	w.registerNode(tn)
	w.Start = tn
	tn.Flow = w
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

// a flow structure is a structureal relationship of nodes and edges that can be rendered
type FlowStruct struct {
	Id    string
	Name  string
	Nodes []Node
	Edges []Edge
}

func (f Workflow) GetStructure() FlowStruct {

	fs := FlowStruct{
		Id:    MakeID(f.Name),
		Name:  f.Name,
		Nodes: make([]Node, 0, 5),
		Edges: make([]Edge, 0, 5),
	}

	for _, n := range f.TaskNodes {
		glog.Info(n)
		fs.Nodes = append(fs.Nodes, Node{Id: MakeID(n.GetName()), Name: n.GetName(), Type: n.GetType()})

		edges := n.GetEdges()
		glog.Info(edges)
		fs.Edges = append(fs.Edges, edges...)
	}

	return fs
}
