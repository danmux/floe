package floe

import (
	"sync"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/par"
)

// Workfloe is the instance of a running floe - it is returned from any FloeFunc which will be invoked by a Launcher
// as many times as there are threads, so there will be an instantiated Workfloe per thread - each one will be constructed with its
// own instances of the nodes.
// Workfloe defines the start and ends and some channels and timers to aide the stepped execution of the floe.
// The tasks themselves know which other task to call. A Workfloe is created and run for each thread of a floe launcher
type Workfloe struct {
	start     *TaskNode       // start is used for a normal workfloe there is only one start and it must be a TaskNode
	end       Node            // any node to end on
	params    *par.Params     // initial condition parameters
	stepper   chan int        // inbound channel - to provide controlled stepping
	taskNodes map[string]Node // map by name of all our registered nodes
	mu        sync.Mutex
	id        string        // copy of id for logging
	triggered bool          // set by the first trigger in the floe - once set all other trigger nodes will not execute
	lcx       launchContext // reference to context in which this floe was launched
}

// NewWorkfloe called by floeFuncs to create a new Workfloe
func NewWorkfloe() *Workfloe {
	return &Workfloe{
		stepper:   make(chan int),
		taskNodes: make(map[string]Node),
	}
}

// AddTaskNode makes a task node with the given name and task and registers it with the Workfloe receiver w
func (w *Workfloe) AddTaskNode(name string, t task.Task) *TaskNode {
	tn := &TaskNode{
		coreNode: newCommonNode(name, w),
		do:       t,
		tType:    t.Type(),
	}
	// add it to the list
	w.registerNode(tn)
	return tn
}

// AddMergeNode creates a new MergeNode with the given name, registers it with the Workfloe w.
func (w *Workfloe) AddMergeNode(name string) *MergeNode {
	mn := &MergeNode{
		coreNode: newCommonNode(name, w),
	}
	w.registerNode(mn)
	return mn
}

// AddTriggerNode creates a new node with the given name and task, registers it with the Workfloe w, and sets its type to trigger
func (w *Workfloe) AddTriggerNode(name string, t task.Task) *TaskNode {
	tn := w.AddTaskNode(name, t)
	tn.tType = "trigger"
	return tn
}

// SetStart marks this node as the starting node for the floe. This will be the first node executed.
func (w *Workfloe) SetStart(tn *TaskNode) {
	w.registerNode(tn)
	w.start = tn
}

// SetEnd marks this node as the last node in the floe if a floe reaches this point then it was probably successful.
func (w *Workfloe) SetEnd(tn Node) {
	w.registerNode(tn)
	w.end = tn
}

// check that the floe has been triggered
func (w *Workfloe) isTriggered() bool {
	w.mu.Lock()
	t := w.triggered
	w.mu.Unlock()
	return t
}

// trigger to make this floe (when it is a trigger floe) as having had one of its triggers nodes fired
func (w *Workfloe) trigger() {
	w.mu.Lock()
	w.triggered = true
	w.mu.Unlock()
}

// exec is the main execute method for a workfloe it finds the special start node and executes it
func (w *Workfloe) exec(p *par.Params) {
	w.params = p
	log.Info("workfloe start", w.id)
	if w.start != nil {
		w.start.Exec(p)
		return
	}
	log.Error("missing start node - make sure you called SetStart passing in a node", w.id)
}

// execTriggers executes any trigger nodes in a trigger floe
func (w *Workfloe) setContext(lcx launchContext) {
	w.lcx = lcx
	for _, n := range w.taskNodes {
		n.setContext(lcx)
	}
}

// execTriggers executes any trigger nodes in a trigger floe
func (w *Workfloe) execTriggers(p *par.Params) {
	w.params = p
	log.Info("starting workfloe triggers", w.id)

	for _, n := range w.taskNodes {
		// launch all none merge nodes
		if n.Type() == "trigger" {
			go n.Exec(p)
		}
	}
}

// registerNode adds this node to the receiver after checking its id is unique
func (w *Workfloe) registerNode(tn Node) {
	if tn.ID() == "" {
		panic("un named node")
	}
	node, in := w.taskNodes[tn.ID()]
	if in {
		// fail loud if they are not actually the same node being registered more than once!
		if node != tn {
			panic("two nodes with the same id " + tn.ID())
		}
	}
	w.taskNodes[tn.ID()] = tn
}

// Structure a flow structure is a structural relationship of nodes and edges that can be rendered in json
type Structure struct {
	ID    string
	Name  string
	Order int
	Nodes []nodeDesc
	Edges []Edge
}

// NodeDesc is a struct for reporting e.g. for json marshalling
type nodeDesc struct {
	ID     string
	Name   string
	Type   string
	Config task.TaskConfig
}

// Structure returns a json marshal-able proxy struct that describes the workfloe and its node graph
func (w *Workfloe) structure(order int) Structure {
	fs := Structure{
		Order: order,
		Nodes: make([]nodeDesc, 0, 5),
		Edges: make([]Edge, 0, 5),
	}

	for _, n := range w.taskNodes {
		fs.Nodes = append(fs.Nodes, nodeDesc{
			ID:     MakeID(n.Name()),
			Name:   n.Name(),
			Type:   n.Type(),
			Config: n.Config(),
		})
		edges := n.Edges()
		fs.Edges = append(fs.Edges, edges...)
	}

	return fs
}
