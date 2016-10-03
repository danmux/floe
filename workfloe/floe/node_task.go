package floe

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/par"
)

// TaskNode task tree structure, implements TriggeredTaskNode
type TaskNode struct {
	coreNode

	tType           string                // the type of task that this node has
	do              task.Task             // this will be the concrete task to execute
	next            map[par.Status][]Node // map of list of next tasks by return code
	usedInMergeNode bool                  // if this is the input to one or more merge nodes
	commandStream   *io.PipeWriter        // the passed in stream - only on thread 0 normally
}

// Type returns the type of this node, alway merge for a merge node. Helps satisfy Node interface.
func (tn *TaskNode) Type() string {
	return tn.tType
}

// SetStream sets up the stream to send output to for this node. Helps satisfy Node interface
func (tn *TaskNode) SetStream(cs *io.PipeWriter) {
	tn.commandStream = cs
}

// Config returns the node config, this will come from the task for this node. Helps satisfy Node interface
func (tn *TaskNode) Config() task.TaskConfig {
	return tn.do.Config()
}

// Edges returns the list of edges to any next nodes. Helps satisfy Node interface
func (tn *TaskNode) Edges() []Edge {
	edges := make([]Edge, 0, 1)
	for val, x := range tn.next {
		for _, xi := range x {
			edges = append(edges, Edge{Name: fmt.Sprintf("%v", val), From: tn.ID(), To: xi.ID()})
		}
	}

	return edges
}

// setMergeTrigger tells this node it is used as a trigger for a merge node
func (tn *TaskNode) setMergeTrigger() {
	tn.usedInMergeNode = true
}

// AddNext this allows fan out - many next tasks can be added to any floe
func (tn *TaskNode) AddNext(forStatus par.Status, t Node) error {
	if tn.do == nil {
		es := "can't add next nodes if current task not set"
		log.Error(es)
		return errors.New(es)
	}

	if tn.floe == nil {
		es := "can't add next nodes if current floe not set"
		log.Error(es)
		return errors.New(es)
	}

	if tn.next == nil {
		tn.next = make(map[par.Status][]Node)
	}

	nextArr, ok := tn.next[forStatus]
	if !ok {
		nextArr = make([]Node, 0, 1)
	}

	nextArr = append(nextArr, t)
	tn.next[forStatus] = nextArr

	return nil
}

// Exec is the main execution function for a node - each node can call Exec on the next appropriate node
func (tn *TaskNode) Exec(inPar *par.Params) {
	log.Info("TaskNode.Exec: ", tn.id)
	if tn.do == nil {
		log.Error("task missing for node")
		return
	}

	// copy the parameters now as these will be the status update
	curPar := &par.Params{}
	if inPar == nil {
		log.Error("ooo - you cant have null parameters")
		return
	}

	// copy the parameters to fill in during this execution
	curPar.Copy(inPar)
	curPar.TaskID = tn.ID()

	// wait for stepper trigger
	<-tn.floe.stepper

	log.Infof("=== Executing id: <%s>", curPar.TaskID)

	// send a not completed signal to mark the start - must copy because receiver may only get the
	// par after the task has finished and marked it complete
	startPar := &par.Params{}
	startPar.Copy(curPar)
	startPar.Complete = false

	// bomb out if stopped
	if !tn.lcx.isRunning() {
		return
	}

	log.Debug("firing start to status chanel")
	tn.lcx.statusChan() <- startPar
	log.Debug("fired start to status chanel")

	// log out the curPar object
	b, _ := json.MarshalIndent(curPar, "", "  ")
	log.Info(string(b))

	// actually execute the task
	tn.do.Exec(tn.lcx.space(), curPar, tn.commandStream)

	// close the stream so the reader can close
	tn.commandStream.Close()

	log.Infof("=== Done  id: <%s> got status: <%d> exitstatus: <%d>", curPar.TaskID, curPar.Status, curPar.ExitStatus)

	// TODO - consider adding all results to the props - for use in later tasks

	// Exec on the task types must be synchronous
	curPar.Complete = true

	// check for the Stop all flag - cos chanel might be closed
	if !tn.lcx.isRunning() {
		log.Warning("thread stopped")
		tn.floe.end.fireDoneChan(curPar) // fire the End done channel if it has not been fired
		return
	}

	if tn.Type() == "trigger" {
		if tn.floe.triggered {
			log.Warning("Trigger fired and ignored as one is already underway")
			return
		}
		// set the flag to ignore all triggers
		log.Warning("Trigger fired - ignoring all others")
		tn.floe.triggered = true
	}

	log.Debug("Firing complete status chanel")
	// fire message to the floe notification channel
	tn.lcx.statusChan() <- curPar

	log.Debug("Fired complete to status chanel")

	if curPar == nil {
		panic("Return parameters cant be nil - at least return the passed in parameters")
	}

	// if this node has a result channel that has not been fired then fire it
	// if this is the end node this will end the floe
	// if this is used as input to a merge node the merge node will record this node as having completed
	log.Debug("Firing to taskNode done chanel")
	tn.fireDoneChan(curPar)
	// if this task feeds into a merge node then firing the done chan is the end of its responsibility
	// the merge node that would be listening for the DoneChan.
	if tn.usedInMergeNode {
		return
	}
	// if this is the special end node then firing the done chan is the end of its responsibility as well
	if tn == tn.floe.end {
		return
	}

	// look for the next set of tasks based on the end status
	log.Infof("Looking up next in a choice of <%d> next nodes, for thread: <%d> with status: <%d> ", curPar.TaskID, len(tn.next), curPar.Status)
	nextTasks, ok := tn.next[curPar.Status]
	// if we did not find a next node matching the end result of this node then if there is only one next node then choose that as a default
	if !ok && len(tn.next) == 1 {
		log.Infof("No matching status task but only one next choice for thread: <%d> with status: <%d> ", curPar.TaskID, curPar.Status)
		for _, n := range tn.next {
			nextTasks = n
			ok = true
			break
		}
	}
	// if we did not find any next tasks then we reached a dead end
	if !ok {
		// so fire the done chanel on this floes end node. That is the signal that this thread has finished.
		log.Warning("unhandled result - this workfloe reached a dead end at id: <%s>, (considered a floe failure)", curPar.TaskID)
		curPar.Status = par.StFail
		curPar.Response = "(" + tn.Name() + ") unhandled result: " + curPar.Response
		tn.floe.end.fireDoneChan(curPar)
		return
	}

	// launch all the the next tasks with the results of this one (TODO - results of all props past?)
	log.Infof("Found %d next task(s) after id: <%s> with status: <%d> ", len(nextTasks), curPar.TaskID, curPar.Status)
	for _, n := range nextTasks {
		go n.Exec(curPar)
	}
}
