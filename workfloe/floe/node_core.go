package floe

import "github.com/floeit/floe/workfloe/par"

// coreNode provides a lot of common node stuff and delivers half of the Node interface.
type coreNode struct {
	id   string        // unique id made from the name but should be html friendly
	name string        // unique name within a floe
	floe *Workfloe     // this node knows which workfloe it is part of
	lcx  launchContext // it also has the interface (that represents the launcher context)

	doneC     chan *par.Params
	doneFired bool
}

func newCommonNode(name string, w *Workfloe) coreNode {
	return coreNode{
		id:    MakeID(name),
		name:  name,
		doneC: make(chan *par.Params, 1), // a buffer of one - as we always send the end even if no one is listening
		floe:  w,
	}
}

// Name returns the name of the node
func (n *coreNode) Name() string {
	return n.name
}

// ID returns the id of the node
func (n *coreNode) ID() string {
	return n.id
}

// fireDoneChan is called whenever this node finished executing. This should be called rather than using doneC directly
// because it is guarded so that it is only fired once. Merge Nodes listen for these events to know when any of its fan in nodes are complete
func (n *coreNode) fireDoneChan(p *par.Params) {
	if n.doneFired {
		return
	}
	n.doneFired = true
	n.doneC <- p
}

// doneChan returns the chanel that once signalled indicates that the task has done executing
func (n *coreNode) doneChan() chan *par.Params {
	return n.doneC
}

func (n *coreNode) setContext(lcx launchContext) {
	n.lcx = lcx
}
