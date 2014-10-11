package customflow

import (
	"floe/tasks"
	f "floe/workflow/flow"
	"fmt"
	"io"
	"time"
)

type SecondTask struct{}

func (ft *SecondTask) Type() string {
	return "second"
}

func (ft *SecondTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	fmt.Println("executing second task")
	p.Response = "second done"
}

type CrapTask struct{}

func (ft *CrapTask) Type() string {
	return "crap"
}

func (ft *CrapTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	fmt.Println("executing crap task")
	time.Sleep(3 * time.Second)
	p.Response = "crap done"
	return
}

type MainFlow struct {
	f.BaseFlow
}

func (l *MainFlow) FlowFunc(threadId int) *f.Workflow {

	w := f.MakeWorkflow(l.Name())
	fn := w.MakeTaskNode("taska", tasks.MakeExecTask("cp", "../src/floe/tasks/loop.sh ."))
	ct := w.MakeTaskNode("task2", tasks.MakeExecTask("./loop.sh", ""))

	// sn := w.MakeTaskNode("task3", tasks.MakeExecTask("ls", "-lrt"))
	sn := w.MakeTaskNode("big clone", tasks.MakeExecTask("git", "clone --progress git@github.com:youtube/vitess.git"))
	// sn := w.MakeTaskNode("task3", tasks.MakeExecTask("./loop.sh", ""))

	// a merge node waits for all triggers to fire before continuing or triggering
	mn := f.MakeMergeNode(w, "task4")
	mn.AddTrigger(fn)
	mn.AddTrigger(sn)
	mn.AddTrigger(ct)

	w.SetStart(fn)

	fn.AddNext(0, ct)

	ct.AddNext(1, sn)
	ct.AddNext(0, sn)

	// the last task must have a channel
	w.SetEnd(mn)

	return w
}

type TestFlow struct {
	f.BaseFlow
}

func (l *TestFlow) FlowFunc(threadId int) *f.Workflow {
	w := f.MakeWorkflow(l.Name())

	// the first node
	fn := w.MakeTaskNode("start", tasks.MakeLsTask("."))

	// the next node
	ct := w.MakeTaskNode("nothing", &CrapTask{})
	// added to the first node for both responses
	fn.AddNext(0, ct)

	// // add a node to the bad path
	sn := w.MakeTaskNode("bad", &SecondTask{})
	ct.AddNext(1, sn)

	fn.AddNext(1, sn)

	// fan out to three parallel tasks if response is 0
	ct1 := w.MakeTaskNode("nothing1", &CrapTask{})
	ct2 := w.MakeTaskNode("nothing2", &CrapTask{})
	ct3 := w.MakeTaskNode("nothing3", &CrapTask{})

	ct.AddNext(0, ct1)
	ct.AddNext(0, ct2)
	ct.AddNext(0, ct3)

	// and wait for all to finish in this merge node
	// a merge node waits for all triggers to fire before continuing or triggering
	mn := f.MakeMergeNode(w, "join")

	// add the three fanned out nodes
	mn.AddTrigger(ct1)
	mn.AddTrigger(ct2)
	mn.AddTrigger(ct3)

	// then end on a final task
	nl := w.MakeTaskNode("end", &SecondTask{})
	mn.SetNext(nl)

	w.SetStart(fn)

	// add a rout from the second node to the end node - so it is not a dead end
	sn.AddNext(0, nl)

	// make the final node explicitly the special end node
	w.SetEnd(nl)

	return w
}

func GetFlows() *f.Project {

	p := f.MakeProject("test project")

	mf := &MainFlow{}
	mf.Init("main flow")
	p.AddFlow(f.MakeFlowLauncher(mf, 1))

	tf := &TestFlow{}
	tf.Init("test flow")
	p.AddFlow(f.MakeFlowLauncher(tf, 1))

	return p
}
