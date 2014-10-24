package tasks

import (
	"bufio"
	f "floe/workflow/flow"
	"fmt"
	"io"
	"os"
	"testing"
)

func Test_Exec(t *testing.T) {

	fl := f.MakeWorkflow("tflow")
	p := f.MakeParams()
	p.Props[f.KEY_WORKSPACE] = "."
	fl.Params = p
	// tsk := MakeExecTask("git", "clone git@github.com:centralway/m-hbci-app.git")

	// tsk := MakeExecTask("ls", "-lrt")
	// tsk := MakeExecTask("pwd", "-L")

	// tsk := MakeExecTask("ls", "")
	tsk := MakeExecTask("./loop.sh", "")

	tn := fl.MakeTaskNode("test", tsk)
	tn.Flow = fl
	r, w := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			fmt.Printf("%s%s \n", "console: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "There was an error with the scanner in attached container", err)
		}
	}()

	tsk.Exec(tn, p, w)

	t.Log("executed")

	if p.Status == 0 {

	} else {
		t.Error("Got bad return status", p.Status)
	}

}
