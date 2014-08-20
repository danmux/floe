package flow

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"third_party/github.com/golang/glog"
)

const (
	SUCCESS = iota
	FAIL
	WORKING
)

const (
	KEY_WORKSPACE = "workspace"
)

type Props map[string]string

type Params struct {
	FlowName   string // these three make up a unique ID for the task
	ThreadId   int
	TaskId     string
	TaskName   string
	Complete   bool // set true on complete tasks
	TaskType   string
	Status     int
	ExitStatus int
	Response   string
	Props      Props
	Raw        []byte
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
	FireTrigger()
	GetEdges() []Edge
	SetStream(*io.PipeWriter)
}

func MakeParams() *Params {
	return &Params{
		Props: Props{KEY_WORKSPACE: "workspace"}, // default workspace name is .... well ... workspace
	}
}

func (p *Params) Copy(ip *Params) {
	// reproduce the id
	p.FlowName = ip.FlowName
	p.ThreadId = ip.ThreadId
	p.TaskName = ip.TaskName
	p.TaskId = ip.TaskId
	p.Complete = false // just to make sure

	// and the other info stuff
	p.TaskType = ip.TaskType
	p.Props = ip.Props
}

// task tree structure
type TaskNode struct {
	Id            string              // unique id made from the name but should be html friendly
	Name          string              // unique name within a flow
	Type          string              // the type of task that this node has
	Flow          *Workflow           // this node knows which workflow it is part of
	C             chan *Params        // the comms/event result chanel - of things to listen to - particularly mergenodes
	do            Task                // this will be the concrete task to execute
	Next          map[int][]*TaskNode // mapped on the return code
	Triggers      bool                // if this triggers one or more merge nodes
	CommandStream *io.PipeWriter      // the passed in stream - only on thread 0 normally
}

func (tn *TaskNode) Trigger() chan *Params {
	return tn.C
}

func (tn *TaskNode) FireTrigger() {
	tn.C <- &Params{}
}

func (tn *TaskNode) GetName() string {
	return tn.Name
}

func (tn *TaskNode) GetType() string {
	return tn.Type
}

func (tn *TaskNode) SetStream(cs *io.PipeWriter) {
	tn.CommandStream = cs
}

func (n *TaskNode) GetEdges() []Edge {
	edges := make([]Edge, 0, 1)
	for val, x := range n.Next {
		for _, xi := range x {
			edges = append(edges, Edge{Name: fmt.Sprintf("%v", val), From: n.Id, To: xi.Id})
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
		es := "can't add next nodes if current task not set"
		glog.Error(es)
		return errors.New(es)
	}

	if tn.Flow == nil {
		es := "can't add next nodes if current flow not set"
		glog.Error(es)
		return errors.New(es)
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
	glog.Info("exec ", tn.Name)
	if tn.do != nil {

		// copy the parameters now as these will be the status update
		curPar := MakeParams()
		if inPar == nil {
			glog.Error("ooo - you cant have null parameters")
			return
		}

		// copy the parameters to fill in during this execution
		curPar.Copy(inPar)
		curPar.TaskName = tn.Name
		curPar.TaskId = tn.Id

		// wait for stepper trigger
		<-tn.Flow.Stepper

		glog.Info("====== Executing >>>>>>>> ", curPar.TaskName, " ", curPar.TaskId, " ", curPar.ThreadId)

		// send a not completed signal to mark the start - must copy because reciever may only get the
		// par after the task has finished and marked it complete
		startPar := MakeParams()
		startPar.Copy(curPar)
		startPar.Complete = false
		tn.Flow.C <- startPar

		// actually execute the task
		tn.do.Exec(tn, curPar, tn.CommandStream)

		glog.Info("===== Done <<<< ", curPar.TaskId, " ", curPar.Status, " ", curPar.ExitStatus, " ", curPar.ThreadId)

		// TODO - consider adding all results to the props - for use in later tasks

		// fire message to the flow notification channel
		curPar.Complete = true
		glog.Info("send")
		tn.Flow.C <- curPar
		glog.Info("sent")

		if curPar == nil {
			panic("Return parameters cant be nil - at least return the passed in parameters")
		}

		// if this node has a result channel fire it - if this is the end node this will end the flow
		if tn.C != nil {
			tn.C <- curPar
		}

		glog.Info("return staus = ", curPar.Status)
		next, ok := tn.Next[curPar.Status]

		// check for the Stop all flag
		if tn.Flow.Stop {
			glog.Warning("thread stopped")
			tn.Flow.End.FireTrigger()
			return
		}

		// if we have another task that matches this return code execute it
		if ok {
			glog.Infof("found %d next task(s)", len(next))
			for _, n := range next {
				go n.Exec(curPar) // launch the next one with the results of this one (TODO - results of all props past?)
			}
		} else {
			// otherwise trigger the last tasks channel - as that is the signal that this thread has finished
			if !tn.Triggers {
				glog.Warning("problem - dead end task - this workflow may never end")
			}
		}

	} else {
		glog.Error("task missing for node")
	}
}

// TODO - make html friendly id
func MakeID(name string) string {
	s := strings.Split(strings.ToLower(name), " ")
	return strings.Join(s, "-")
}

type Task interface {
	// exec fills in and returns the params
	Exec(t *TaskNode, p *Params, out *io.PipeWriter)
	Type() string
}
