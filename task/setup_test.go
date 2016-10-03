package task

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/floeit/floe/workfloe/par"
)

// type testFloe struct {
// 	f.BaseLaunchable
// }

// func (fl *testFloe) FloeFunc(threadId int) *f.Workfloe {
// 	w := f.NewWorkfloe()
// 	p := &f.Params{}
// 	w.Params = p
// 	return w
// }

// func (fl *testFloe) GetProps() *f.Props {
// 	p := fl.DefaultProps()
// 	(*p)[f.KeyTidyDesk] = "keep"
// 	return p
// }

// func (fl *testFloe) ID() string {
// 	return "test"
// }

// func setup() *f.Workfloe {
// 	p := f.NewProject()
// 	p.SetRoot("~/floetest")
// 	p.SetName("test")
// 	tf := &testFloe{}
// 	fl := p.AddLauncher(tf, 1, nil, nil)
// 	fl.TidyDeskPolicy(*tf.GetProps())

// 	w := fl.NewWorkfloe(0)
// 	return w
// }

func ExecFrame(t *testing.T, tsk Task, expected int) *par.Params {

	curPar := &par.Params{
		Props: map[string]string{},
	}

	pr, pw := io.Pipe()

	var output []string
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			t := scanner.Text()
			output = append(output, t)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "There was an error with the scanner in attached container", err)
		}
	}()

	// make a temp workspace in echo $TMPDIR
	td := filepath.Join(os.TempDir(), "floe_test")
	if err := os.RemoveAll(td); err != nil { // remove current directory
		t.Fatal(err)
	}
	if err := os.MkdirAll(td, 0777); err != nil {
		t.Fatal(err)
	}

	// add a file...
	d1 := []byte("hello\ngo\n")
	if err := ioutil.WriteFile(filepath.Join(td, "testy_file.txt"), d1, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &Context{
		WorkspacePath: td,
	}

	tsk.Exec(ctx, curPar, pw) // TODO not nil

	t.Log("ExecFrame complete")

	if curPar.Status != 0 {
		t.Error("Got bad return status", curPar.Status)
	} else {
		t.Log(strings.Join(output, "\n"))
	}

	if len(output) != expected {
		t.Errorf("output should have been contained: %d lines but was: %d", expected, len(output))
	}

	return curPar
}
