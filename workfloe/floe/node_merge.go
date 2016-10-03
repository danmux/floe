package floe

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/par"
)

// MergeNode is a type of task node that simply waits for all of its registered tasks to complete (trigger) before it completes.
type MergeNode struct {
	coreNode

	next   *TaskNode       // there can only be one next for a merge node
	group  sync.WaitGroup  // the sync group that once done indicates that all input trigger nodes worked
	inputs map[string]Node // the nodes that this merge node waits for
}

// Type returns the type of this node, alway merge for a merge node. Helps satisfy Node interface.
func (tn *MergeNode) Type() string {
	return "merge"
}

// SetStream is unimplemented for a merge node. Helps satisfy Node interface
func (tn *MergeNode) SetStream(cs *io.PipeWriter) {}

// Config returns the node config. Helps satisfy Node interface
func (tn *MergeNode) Config() task.TaskConfig {
	return task.TaskConfig{
		Command: "wait for all inputs",
	}
}

// Edges returns the list of edges including attached triggers and the next node. Helps satisfy Node interface
func (tn *MergeNode) Edges() []Edge {
	edges := make([]Edge, 0, 1)
	for _, x := range tn.inputs { // triggers are inbound
		edges = append(edges, Edge{Name: "0", From: x.ID(), To: tn.ID()})
	}

	if tn.next != nil {
		edges = append(edges, Edge{Name: fmt.Sprintf("%v", 0), From: tn.ID(), To: tn.next.ID()})
	}

	return edges
}

// Exec merge nodes don't execute they sit there waiting for inbound triggers
func (tn *MergeNode) Exec(p *par.Params) {}

// SetNext adds the next node for this MergeNode.
func (tn *MergeNode) SetNext(t *TaskNode) {
	// make sure we have a cpy of this in the parent map
	tn.floe.registerNode(t)
	tn.next = t
}

// AddTrigger adds a node that this MergeNode will wait for completion on
func (tn *MergeNode) AddTrigger(t Node) error {

	if tn.floe == nil {
		return errors.New("can't add next nodes if current floe not set")
	}

	if tn.inputs == nil {
		tn.inputs = make(map[string]Node)
	}

	// need to know if its the first node because we can start the goroutine that waits for all inputs to fire
	first := false
	if len(tn.inputs) == 0 {
		first = true
	}

	// make sure this task has a chanel
	if t.doneChan() == nil {
		panic("triggers must have a done chanel")
	}

	// tell the node it is the input to a merge node
	t.setMergeTrigger()

	// check that we don't already have this node as a trigger
	_, ok := tn.inputs[t.ID()]
	if ok {
		return nil
	}
	tn.inputs[t.ID()] = t

	// last params wins - copied into the merge nodes curPar when the merge node is done
	var donePar *par.Params

	tn.group.Add(1)
	go func() {
		donePar = <-t.doneChan() // wait for the singe done event for this node
		tn.group.Done()
	}()

	// if this is not the first trigger added then we are all done
	if !first {
		return nil
	}

	// for the first added trigger start the goroutine that waits for all triggers to have fired
	go func() {
		tn.group.Wait()
		// tell the floe status channel we have completed
		curPar := &par.Params{}
		curPar.Copy(donePar)
		curPar.TaskID = tn.ID()
		curPar.Complete = true
		// tell the launcher
		tn.lcx.statusChan() <- curPar
		log.Debug("MergeNode channel fired")
		// and trigger our end channel
		tn.fireDoneChan(curPar)
		// if the launcher is not running then be done
		if !tn.lcx.isRunning() {
			return
		}
		// if there is no further task then
		if tn.next == nil {
			return
		}
		log.Debugf("Merge node %s calling single next node", tn.ID())
		go tn.next.Exec(curPar)
	}()

	return nil
}

// does nothing for a trigger node but helps satisfy Node interface
func (tn *MergeNode) setMergeTrigger() {}
