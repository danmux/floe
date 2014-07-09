package customflow

import (
	"floe/tasks"
	f "floe/workflow/flow"
	"fmt"
)

type SecondTask struct{}

func (ft *SecondTask) Type() string {
	return "second"
}

func (ft *SecondTask) Exec(t *f.TaskNode, p *f.Params) *f.Params {
	fmt.Println("executing second task")
	p.Response = "second done"
	return p
}

type CrapTask struct{}

func (ft *CrapTask) Type() string {
	return "crap"
}

func (ft *CrapTask) Exec(t *f.TaskNode, p *f.Params) *f.Params {
	fmt.Println("executing crap task")
	p.Response = "crap done"
	return p
}

func GetMainFlow(threadId int) *f.Workflow {
	w := f.MakeWorkflow("main flow")
	fn := f.MakeTaskNode("start", tasks.MakeLsTask("."))
	sn := f.MakeTaskNode("ls -lrt", tasks.MakeExecTask("git", "clone git@github.com:centralway/m-api.git"))
	ct := f.MakeTaskNode("something", &CrapTask{})

	// a merge node waits for all triggers to fire before continuing or triggering
	mn := f.MakeMergeNode(w, "end")
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

func GetTestFlow(threadId int) *f.Workflow {
	w := f.MakeWorkflow("test flow")
	fn := f.MakeTaskNode("start", tasks.MakeLsTask("."))
	sn := f.MakeTaskNode("bad", &SecondTask{})
	ct := f.MakeTaskNode("nothing", &CrapTask{})

	ct1 := f.MakeTaskNode("nothing1", &CrapTask{})

	ct2 := f.MakeTaskNode("nothing2", &CrapTask{})

	ct3 := f.MakeTaskNode("nothing3", &CrapTask{})

	nl := f.MakeTaskNode("end", &SecondTask{})

	// a merge node waits for all triggers to fire before continuing or triggering
	mn := f.MakeMergeNode(w, "join")

	mn.AddTrigger(ct1)
	mn.AddTrigger(ct2)
	mn.AddTrigger(ct3)

	mn.SetNext(nl)

	w.SetStart(fn)

	fn.AddNext(0, ct)

	ct.AddNext(1, sn)
	ct.AddNext(0, ct1)
	ct.AddNext(0, ct2)
	ct.AddNext(0, ct3)

	sn.AddNext(0, nl)
	// the last task must have a channel
	w.SetEnd(nl)

	return w
}

func GetFlows() *f.Project {
	fmt.Println("creating flows")

	p := f.MakeProject("test project")

	name := "main launcher"
	p.Flows[name] = f.MakeFlowLauncher(name, GetMainFlow, 1)

	// name = "test launcher"
	// p.Flows[name] = f.MakeFlowLauncher(name, GetTestFlow, 200)

	return p
}
