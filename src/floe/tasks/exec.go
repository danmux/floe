package tasks

import (
	f "floe/workflow/flow"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ExecTask struct {
	cmd  string
	args string
}

func (ft ExecTask) Type() string {
	return "execute"
}

func MakeExecTask(cmd, args string) ExecTask {
	return ExecTask{
		cmd:  cmd,
		args: args,
	}
}

func (ft ExecTask) Exec(t *f.TaskNode, p *f.Params) *f.Params {
	fmt.Println("executing command")

	cmd, ok := p.Props["cmd"]
	// if no passed in cmd use defualt
	if !ok {
		cmd = ft.cmd
	}

	if cmd == "" {
		p.Status = f.FAIL
		p.Response = "no cmd specified"
		return p
	}

	args, ok := p.Props["args"]
	// if no passed in args use defualt
	if !ok {
		args = ft.args
	}

	fmt.Println(cmd, args)

	ars := strings.Split(args, " ")

	eCmd := exec.Command(cmd, ars...)

	// this is mandatory
	eCmd.Dir = t.Flow.Params.Props["workspace"]

	sout, err := eCmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		p.Status = 1
		return p
	}
	eout, err := eCmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		p.Status = 1
		return p
	}

	err = eCmd.Start()
	if err != nil {
		fmt.Println(err)
		p.Status = 1
		return p
	}

	io.Copy(os.Stdout, sout)
	io.Copy(os.Stdout, eout)

	eCmd.Wait()

	p.Response = "exec command done"
	p.Status = f.SUCCESS

	return p
}
