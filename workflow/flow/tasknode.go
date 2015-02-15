package flow

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"strings"
)

const (
	SUCCESS = iota
	FAIL
	WORKING
	LOOP
)

const (
	KEY_WORKSPACE = "workspace"       // folder for per project files
	KEY_TIDY_DESK = "reset_workspace" // reset or keep
	KEY_TRIGGERS  = "triggers"        // folder for trigger state
)

// the interface that the tasknodes hod that actually do the work
// these task types are added to floe/tasks
type Task interface {
	// exec fills in and returns the params
	Exec(t *TaskNode, p *Params, out *io.PipeWriter)
	Type() string
	Config() TaskConfig // json representation of the config for the node
}

// TaskConfig holds any usefull public information about a specific task
// at the moment this is just used for display
type TaskConfig struct {
	Command string
}

// the interface for all nodes in a flow
type TriggeredTaskNode interface {
	// exec fills in and returns the params
	Exec(p *Params)
	WorkFlow() *Workflow
	Type() string
	Name() string
	Id() string
	DoneChan() chan *Params
	FireDoneChan(p *Params)
	Edges() []Edge
	SetStream(*io.PipeWriter)
	SetWorkFlow(*Workflow)
	SetMergeTrigger()
	Config() TaskConfig
}

// task tree structure
type TaskNode struct {
	id              string                      // unique id made from the name but should be html friendly
	name            string                      // unique name within a flow
	tType           string                      // the type of task that this node has
	flow            *Workflow                   // this node knows which workflow it is part of
	C               chan *Params                // the comms/event result channel only triggered when task complete - mergenodes particularly like this
	do              Task                        // this will be the concrete task to execute
	Next            map[int][]TriggeredTaskNode // map of tasks by return code
	usedInMergeNode bool                        // if this is the input to one or more merge nodes
	CommandStream   *io.PipeWriter              // the passed in stream - only on thread 0 normally
}

func (tn *TaskNode) SetWorkFlow(f *Workflow) {
	tn.flow = f
}

func (tn *TaskNode) WorkFlow() *Workflow {
	return tn.flow
}

func (tn *TaskNode) SetMergeTrigger() {
	tn.usedInMergeNode = true
}

func (tn *TaskNode) DoneChan() chan *Params {
	return tn.C
}

func (tn *TaskNode) FireDoneChan(p *Params) {
	tn.C <- p
}

func (tn *TaskNode) Name() string {
	return tn.name
}

func (tn *TaskNode) Id() string {
	return tn.id
}

func (tn *TaskNode) Type() string {
	return tn.tType
}

func (tn *TaskNode) SetStream(cs *io.PipeWriter) {
	tn.CommandStream = cs
}

func (tn *TaskNode) Config() TaskConfig {
	return tn.do.Config()
}

func (n *TaskNode) Edges() []Edge {
	edges := make([]Edge, 0, 1)
	for val, x := range n.Next {
		for _, xi := range x {
			edges = append(edges, Edge{Name: fmt.Sprintf("%v", val), From: n.Id(), To: xi.Id()})
		}
	}

	return edges
}

func (tn *TaskNode) SetTask(t Task) {
	tn.do = t
	// update the node with this task type
	tn.tType = tn.do.Type()
}

// this allows fan out - many next tasks can be added to any flow
func (tn *TaskNode) AddNext(forStatus int, t TriggeredTaskNode) error {
	if tn.do == nil {
		es := "can't add next nodes if current task not set"
		glog.Error(es)
		return errors.New(es)
	}

	if tn.flow == nil {
		es := "can't add next nodes if current flow not set"
		glog.Error(es)
		return errors.New(es)
	}

	if tn.Next == nil {
		tn.Next = make(map[int][]TriggeredTaskNode)
	}

	// thell the task what flow it is in
	if t.WorkFlow() != tn.flow {
		panic("next nodes have to be in the same workflow")
	}

	nextArr, ok := tn.Next[forStatus]
	if !ok {
		nextArr = make([]TriggeredTaskNode, 0, 1)
	}

	// make sure we have a cpy of this in the parent map
	tn.flow.registerNode(t)

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
		curPar.TaskName = tn.Name()
		curPar.TaskId = tn.Id()

		// wait for stepper trigger
		<-tn.flow.Stepper

		glog.Info("====== Executing >>>>>>>> ", curPar.TaskName, " ", curPar.TaskId, " ", curPar.ThreadId)

		// send a not completed signal to mark the start - must copy because reciever may only get the
		// par after the task has finished and marked it complete
		startPar := MakeParams()
		startPar.Copy(curPar)
		startPar.Complete = false
		tn.flow.C <- startPar

		// log out the curPar object
		b, _ := json.MarshalIndent(curPar, "", "  ")
		glog.Info(string(b))

		// actually execute the task
		tn.do.Exec(tn, curPar, tn.CommandStream)

		glog.Info("===== Done <<<< ", curPar.TaskId, " ", curPar.Status, " ", curPar.ExitStatus, " ", curPar.ThreadId)

		// TODO - consider adding all results to the props - for use in later tasks

		// Exec on the task types must be synchronous
		curPar.Complete = true

		// check for the Stop all flag - cos chanel might be closed
		if tn.flow.Stop {
			glog.Warning("thread stopped")
			tn.flow.End.FireDoneChan(curPar)
			return
		}

		if tn.Type() == "trigger" {
			if tn.flow.IgnoreTriggers {
				glog.Warning("trigger fired and ignored as one is already underway")
				return
			}
			// set the flag to ignore all triggers
			glog.Warning("trigger fired - ignoring all others")
			tn.flow.IgnoreTriggers = true
		}

		// fire message to the flow notification channel
		tn.flow.C <- curPar

		if curPar == nil {
			panic("Return parameters cant be nil - at least return the passed in parameters")
		}

		// if this node has a result channel fire it - if this is the end node this will end the flow
		if tn.C != nil {
			// if chanel backed up clear it
			if len(tn.C) > 0 {
				<-tn.C
			}
			tn.C <- curPar
		}

		glog.Info("return staus = ", curPar.Status)
		next, ok := tn.Next[curPar.Status]

		// if we have another task that matches this return code execute it
		if ok {
			glog.Infof("found %d next task(s)", len(next))
			for _, n := range next {
				go n.Exec(curPar) // launch the next one with the results of this one (TODO - results of all props past?)
			}
		} else {
			// otherwise trigger the last tasks channel - as that is the signal that this thread has finished
			if !tn.usedInMergeNode && tn != tn.flow.End {
				glog.Warning("problem - dead end task - this workflow may not of ended properly")
				tn.flow.End.FireDoneChan(curPar)
			}
		}

	} else {
		glog.Error("task missing for node")
	}
}

// TODO - make html friendly id
func MakeID(name string) string {

	s := strings.Split(strings.ToLower(strings.TrimSpace(name)), " ")
	return strings.Join(s, "-")
}

// struct for reporting e.g. for json-ifying
type Node struct {
	Id     string
	Name   string
	Type   string
	Config TaskConfig
}

// struct for reporting e.g. for json-ifying
type Edge struct {
	Name string
	From string
	To   string
}
