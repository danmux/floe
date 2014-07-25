package flow

import (
	"fmt"
)

type Workflow struct {
	Name      string
	Start     *TaskNode
	End       TriggeredTaskNode
	Params    *Params                      // initial condition parameters
	C         chan *Params                 // the chanel that tasknodes attach to and send status updates
	Stepper   chan int                     // inbound channel - to provide stepping
	TaskNodes map[string]TriggeredTaskNode // map by name of all our nodes
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
	fmt.Println("workflow start", w.Name)
	if w.Start != nil {
		w.Start.Exec(p)
		return
	}

	fmt.Println("no task")
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
		fs.Nodes = append(fs.Nodes, Node{Id: MakeID(n.GetName()), Name: n.GetName(), Type: n.GetType()})

		edges := n.GetEdges()
		fs.Edges = append(fs.Edges, edges...)
	}

	return fs
}
