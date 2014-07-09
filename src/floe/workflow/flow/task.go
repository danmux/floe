package flow

import (
	"errors"
	"fmt"
	"strings"
)

const (
	SUCCESS = iota
	FAIL
	WORKING
)

type Props map[string]string

type Params struct {
	FlowName string // these three make up a unique ID for the task
	ThreadId int
	TaskName string

	TaskType string
	Status   int
	Response string
	Props    Props
	Raw      []byte
}

// descriptive stuff for json-ifying
type Node struct {
	Id   string
	Name string
	Type string
}

type Edge struct {
	Name string
	From string
	To   string
}

type TriggeredTaskNode interface {
	// exec fills in and returns the params
	Exec(p *Params)
	GetType() string
	GetName() string
	Trigger() chan *Params
	GetEdges() []Edge
}

func MakeParams() *Params {
	return &Params{
		Props: make(Props),
	}
}

func (p *Params) Copy(ip *Params) {
	// reproduce the id
	p.FlowName = ip.FlowName
	p.ThreadId = ip.ThreadId
	p.TaskName = ip.TaskName

	// and the other info stuff
	p.TaskType = ip.TaskType
	p.Props = ip.Props
}

// task tree structure
type TaskNode struct {
	Name string // unique name within a flow
	Type string // the type of task that this node has
	Flow *Workflow
	C    chan *Params
	do   Task
	Next map[int][]*TaskNode // mapped on the return code
}

func MakeTaskNode(name string, t Task) *TaskNode {
	tn := &TaskNode{
		Name: name,
		C:    make(chan *Params),
	}

	tn.SetTask(t)

	return tn
}

func (tn *TaskNode) Trigger() chan *Params {
	return tn.C
}

func (tn *TaskNode) GetName() string {
	return tn.Name
}

func (tn *TaskNode) GetType() string {
	return tn.Type
}

func (n *TaskNode) GetEdges() []Edge {
	edges := make([]Edge, 0, 1)
	for val, x := range n.Next {
		for _, xi := range x {
			edges = append(edges, Edge{Name: fmt.Sprintf("%v", val), From: n.Name, To: xi.Name})
		}
	}

	return edges
}

func (tn *TaskNode) SetTask(t Task) {
	tn.do = t
	// update the node with this task type
	tn.Type = tn.do.Type()
}

func (tn *TaskNode) AddNext(forStatus int, t *TaskNode) error {
	if tn.do == nil {
		return errors.New("can't add next nodes if current task not set")
	}

	if tn.Flow == nil {
		return errors.New("can't add next nodes if current flow not set")
	}

	if tn.Next == nil {
		tn.Next = make(map[int][]*TaskNode)
	}

	// add it to the flow - this allows fan out - many next tasks can be added to any flow
	t.Flow = tn.Flow

	nextArr, ok := tn.Next[forStatus]
	if !ok {
		nextArr = make([]*TaskNode, 0, 1)
	}

	// make sure we have a cpy of this in the parent map
	tn.Flow.registerNode(t)

	nextArr = append(nextArr, t)
	tn.Next[forStatus] = nextArr

	return nil
}

func (tn *TaskNode) Exec(inPar *Params) {
	fmt.Println("task node exec", tn.Name)
	if tn.do != nil {

		// copy the parameters now as these will be the status update
		newPar := MakeParams()
		if inPar == nil {
			fmt.Println("Booo - you cant have null parameters")
			return
		}
		newPar.Copy(inPar)

		<-tn.Flow.Stepper

		// actually execute the task
		retPar := tn.do.Exec(tn, newPar)

		if retPar == nil {
			panic("Return parameters cant be nil - at least return the passed in parameters")
		}

		// TODO - consider adding all results to the props - for use in later tasks

		// fire message to flow notification channel
		tn.Flow.C <- retPar

		// if this node has a result channel fire it
		if tn.C != nil {
			tn.C <- retPar
		}

		fmt.Println("    result = ", retPar.Status)
		next, ok := tn.Next[retPar.Status]
		// if we have another task that matches this return code execute it
		if ok {
			for _, n := range next {
				go n.Exec(retPar) // launch the next one with the results of this one (TODO - results of all props past?)
			}
		} else {
			// otherwise trigger the last tasks channel - as that is the signal that this thread has finished
			tn.Flow.End.Trigger() <- retPar
		}
	}
}

func MakeID(name string) string {
	s := strings.Split(strings.ToLower(name), " ")
	return strings.Join(s, "-")
}

type Task interface {
	// exec fills in and returns the params
	Exec(t *TaskNode, p *Params) *Params
	Type() string
}
