package tasks

import (
	f "floe/workflow/flow"
	"testing"
)

func Test_Exec(t *testing.T) {

	fl := f.MakeWorkflow("tflow")
	p := f.MakeParams()
	p.Props["workspace"] = "."
	fl.Params = p
	// tsk := MakeExecTask("git", "clone git@github.com:centralway/m-hbci-app.git")

	tsk := MakeExecTask("ls", "-lrt")

	tn := f.MakeTaskNode("test", tsk)
	tn.Flow = fl
	tsk.Exec(tn, p)

}
