package testfloe

import (
	"time"

	"github.com/floeit/floe/task"
	f "github.com/floeit/floe/workfloe/floe"
	"github.com/floeit/floe/workfloe/par"
)

const baseTime = time.Millisecond * 10

type buildFloe struct {
	f.BaseLaunchable
	repo   string
	branch string
}

func (l *buildFloe) GetProps() *par.Props {
	p := l.DefaultProps()
	(*p)[par.KeyTidyDesk] = "keep" // this to not trash the workspace
	return p
}

func (l *buildFloe) FloeFunc(threadId int) *f.Workfloe {

	w := f.NewWorkfloe()

	updateWs, build := buildWorkspace(w, nil, l.repo, l.branch)

	kill := w.AddTaskNode("killprocs", task.MakeDelayTask(1*baseTime))

	build.AddNext(0, kill)

	unit := w.AddTaskNode("unit", task.MakeDelayTask(1*baseTime))

	launch := w.AddTaskNode("launch", task.MakeDelayTask(1*baseTime))

	kill.AddNext(0, unit)
	kill.AddNext(1, unit) // may well fail if no m-be processes are running

	unit.AddNext(0, launch)

	pause := w.AddTaskNode("settle", task.MakeDelayTask(2*baseTime))

	// export GOPATH=${PWD}; go install ./src/...
	buildConTest := w.AddTaskNode("build con test", task.MakeDelayTask(1*baseTime))
	runConTest := w.AddTaskNode("run con test", task.MakeDelayTask(1*baseTime))

	buildIdTest := w.AddTaskNode("build id test", task.MakeDelayTask(1*baseTime))
	runIdTest := w.AddTaskNode("run id test", task.MakeDelayTask(1*baseTime))

	buildApiTest := w.AddTaskNode("build api test", task.MakeDelayTask(1*baseTime))
	runApiTest := w.AddTaskNode("run api test", task.MakeDelayTask(1*baseTime))

	launch.AddNext(0, pause)
	pause.AddNext(0, buildConTest)
	pause.AddNext(0, buildIdTest)
	pause.AddNext(0, buildApiTest)

	buildConTest.AddNext(0, runConTest)
	buildIdTest.AddNext(0, runIdTest)
	buildApiTest.AddNext(0, runApiTest)

	mn := w.AddMergeNode("wait tests")

	mn.AddTrigger(runConTest)
	mn.AddTrigger(runIdTest)
	mn.AddTrigger(runApiTest)

	done := w.AddTaskNode("done", task.MakeDelayTask(0))

	mn.SetNext(done)

	w.SetStart(updateWs)
	w.SetEnd(done)

	return w
}

// checkout workspace and build it
func buildWorkspace(w *f.Workfloe, start *f.TaskNode, repo, branch string) (begin *f.TaskNode, end *f.TaskNode) {

	updateWs := w.AddTaskNode("update workspace", task.MakeDelayTask(1*baseTime))

	update := w.AddTaskNode("update", task.MakeDelayTask(1*baseTime))

	clean := w.AddTaskNode("clean", task.MakeDelayTask(1*baseTime))

	clone := w.AddTaskNode("clone workspace", task.MakeDelayTask(1*baseTime))

	create := w.AddTaskNode("create repos", task.MakeDelayTask(1*baseTime))

	switchBranch := w.AddTaskNode("switch", task.MakeDelayTask(1*baseTime))

	build := w.AddTaskNode("build", task.MakeDelayTask(1*baseTime))

	if start != nil {
		start.AddNext(0, updateWs)
	}

	updateWs.AddNext(1, update)
	updateWs.AddNext(0, clean)
	clean.AddNext(0, clone)
	clean.AddNext(1, clone) // if clean failed it was already deleted so carry on regardless
	clone.AddNext(0, create)
	create.AddNext(0, switchBranch)

	switchBranch.AddNext(0, build)
	update.AddNext(0, build)
	update.AddNext(1, clean)

	return updateWs, build

}
