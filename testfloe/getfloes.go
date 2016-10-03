package testfloe

import (
	"time"

	"github.com/floeit/floe/task"
	"github.com/floeit/floe/trigger"
	floe "github.com/floeit/floe/workfloe/floe"
)

func getTestFloes(p *floe.Project) {

	// set some properties on our floe object
	fl := &buildFloe{
		repo:   "danmux",
		branch: "development",
	}
	fl.Init("Test Build")
	// p.AddLauncher(fl, 1, nil, testTriggerFloe)
	p.AddLauncher(fl, 1, nil, nil)
}

func GetFloes(p *floe.Project, env string) {

	p.SetName("V3")

	if env == "test" {
		getTestFloes(p)
	}
}

func testTriggerFloe(threadId int) *floe.Workfloe {
	w := floe.NewWorkfloe()

	// t1 := w.MakeTriggerNode("wait 14", trigger.MakeDelayTrigger(14*time.Second))
	t2 := w.AddTriggerNode("wait 2", trigger.MakeDelayTrigger(2*time.Second))

	// hip_start := w.MakeTaskNode("ping hipchat", task.MakeDelayTask(5*time.Second))

	// co := w.MakeTaskNode("git checkout", task.MakeDelayTask(15*time.Second))

	last := w.AddTaskNode("finish", task.MakeDelayTask(5*time.Second))

	// tpush := w.MakeTriggerNode("push floeit pages", trigger.MakeGitPushTrigger("git@github.com:floeit/floeit.github.io.git", "", 10))

	// hip_start.AddNext(0, co)
	// co.AddNext(0, last)

	// t1.AddNext(0, co)
	t2.AddNext(0, last)

	// tpush.AddNext(0, co)

	w.SetEnd(last)

	return w
}
