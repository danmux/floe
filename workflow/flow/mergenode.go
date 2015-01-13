package flow

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// A merge node is a type of task that simply waits for all of its registered tasks to complete (trigger)
type MergeNode struct {
	name     string
	id       string
	flow     *Workflow
	C        chan *Params
	Next     *TaskNode
	Triggers map[string]TriggeredTaskNode
	Group    sync.WaitGroup
	first    bool
}

func (tn *MergeNode) SetMergeTrigger() {
}

func (tn *MergeNode) SetWorkFlow(f *Workflow) {
	tn.flow = f
}

func (tn *MergeNode) WorkFlow() *Workflow {
	return tn.flow
}

func (tn *MergeNode) DoneChan() chan *Params {
	return tn.C
}

func (tn *MergeNode) FireDoneChan(p *Params) {
	tn.C <- p
}

func (tn *MergeNode) Name() string {
	return tn.name
}

func (tn *MergeNode) Id() string {
	return tn.id
}

func (tn *MergeNode) Type() string {
	return "merge"
}

func (tn *MergeNode) SetStream(cs *io.PipeWriter) {}

func (tn *MergeNode) Edges() []Edge {
	edges := make([]Edge, 0, 1)
	for _, x := range tn.Triggers { // triggers are inbound
		edges = append(edges, Edge{Name: "0", From: x.Id(), To: tn.Id()})
	}

	if tn.Next != nil {
		edges = append(edges, Edge{Name: fmt.Sprintf("%v", 0), From: tn.Id(), To: tn.Next.Id()})
	}

	return edges
}

// mergenodes only lissten for triggers
func (tn *MergeNode) Exec(p *Params) {}

func (tn *MergeNode) SetNext(t *TaskNode) {
	// make sure we have a cpy of this in the parent map
	tn.flow.registerNode(t)
	tn.Next = t
}

func (tn *MergeNode) AddTrigger(t TriggeredTaskNode) error {

	if tn.flow == nil {
		return errors.New("can't add next nodes if current flow not set")
	}

	if tn.Triggers == nil {
		tn.Triggers = make(map[string]TriggeredTaskNode)
	}

	// make sure this task has a chanel
	if t.DoneChan() == nil {
		panic("triggers must have a done chanel")
	}

	if t.WorkFlow() != tn.flow {
		panic("triggers must be in the same workflow as the merge node they trigger")
	}

	// tell the tasknode it does trigger something
	t.SetMergeTrigger()

	_, ok := tn.Triggers[t.Name()]
	if !ok {
		tn.Triggers[t.Name()] = t

		var par *Params // last params wins
		tn.Group.Add(1)
		go func() {
			par = <-t.DoneChan()
			tn.Group.Done()
		}()

		if tn.first {
			go func() {
				tn.Group.Wait()
				// tell the flow status channel we have completed
				curPar := &Params{}
				curPar.Copy(par)
				curPar.TaskName = tn.Name()
				curPar.TaskId = tn.Id()

				curPar.Complete = true
				tn.flow.C <- curPar
				fmt.Println("Trigger fired")

				// and trigger our end channel
				tn.C <- curPar

				// and if there is another task fire that off
				if tn.Next != nil {
					fmt.Println("Merge node calling single next node")
					go tn.Next.Exec(curPar)
				}
			}()
		}
		tn.first = false
	}
	return nil
}
