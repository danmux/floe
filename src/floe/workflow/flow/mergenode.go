package flow

import (
	"errors"
	"fmt"
	"sync"
)

// A merge node is a type of task that simply waits for all of its registered tasks to complete (trigger)
type MergeNode struct {
	Name     string
	Id       string
	Flow     *Workflow
	C        chan *Params
	Next     *TaskNode
	Triggers map[string]*TaskNode
	Group    sync.WaitGroup
	first    bool
}

func MakeMergeNode(fl *Workflow, name string) *MergeNode {
	mn := &MergeNode{
		Flow:  fl,
		Name:  name,
		Id:    MakeID(name),
		first: true,
		C:     make(chan *Params, 1), // a buffer of one - as we always send the end even if no one is listening
	}
	fl.registerNode(mn)
	return mn
}

func (tn *MergeNode) Trigger() chan *Params {
	return tn.C
}

func (tn *MergeNode) GetName() string {
	return tn.Name
}

func (tn *MergeNode) GetType() string {
	return "merge"
}

func (tn *MergeNode) GetEdges() []Edge {
	edges := make([]Edge, 0, 1)
	for _, x := range tn.Triggers { // triggers are inbound
		edges = append(edges, Edge{Name: "", From: x.Name, To: tn.Name})
	}

	if tn.Next != nil {
		edges = append(edges, Edge{Name: fmt.Sprintf("%v", 0), From: tn.Name, To: tn.Next.Name})
	}

	return edges
}

// mergenodes only lissten for triggers
func (tn *MergeNode) Exec(p *Params) {}

func (tn *MergeNode) SetNext(t *TaskNode) {
	// make sure we have a cpy of this in the parent map
	tn.Flow.registerNode(t)
	tn.Next = t
}

func (tn *MergeNode) AddTrigger(t *TaskNode) error {

	if tn.Flow == nil {
		return errors.New("can't add next nodes if current flow not set")
	}

	if tn.Triggers == nil {
		tn.Triggers = make(map[string]*TaskNode)
	}

	// make sure this task has a chanel
	if t.C == nil {
		t.C = make(chan *Params)
	}

	// add it to the flow
	t.Flow = tn.Flow

	// tell the tasknode it does trigger something
	t.Triggers = true

	_, ok := tn.Triggers[t.Name]
	if !ok {
		tn.Triggers[t.Name] = t

		var par *Params // last params wins
		tn.Group.Add(1)
		go func() {
			par = <-t.C
			tn.Group.Done()
		}()

		if tn.first {
			go func() {
				tn.Group.Wait()
				// tell the flow status channel we have completed
				curPar := &Params{}
				curPar.Copy(par)
				curPar.TaskName = tn.Name
				curPar.TaskId = tn.Id

				tn.Flow.C <- curPar
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
