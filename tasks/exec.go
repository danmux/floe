package tasks

import (
	f "floe/workflow/flow"
	"io"
	"os/exec"
	// "strings"
	"syscall"
	"third_party/github.com/golang/glog"
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

func (ft ExecTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing command")

	cmd, ok := p.Props["cmd"]
	// if no passed in cmd use defualt
	if !ok {
		cmd = ft.cmd
	}

	if cmd == "" {
		p.Status = f.FAIL
		p.Response = "no cmd specified"
		return
	}

	args, ok := p.Props["args"]
	// if no passed in args use defualt
	if !ok {
		args = ft.args
	}

	glog.Info("cmd: ", cmd, " args: >", args, "<")
	argstr := cmd + " " + args

	eCmd := exec.Command("bash", "-c", argstr)

	// this is mandatory
	eCmd.Dir = t.Flow.Params.Props[f.KEY_WORKSPACE]
	glog.Info("working directory: ", eCmd.Dir)

	out.Write([]byte(eCmd.Dir + "$ " + argstr + "\n\n"))

	var err error
	if out != nil {
		sout, err := eCmd.StdoutPipe()
		if err != nil {
			glog.Info(err)
			p.Status = f.FAIL
			return
		}
		eout, err := eCmd.StderrPipe()
		if err != nil {
			glog.Error(err)
			p.Status = f.FAIL
			return
		}

		glog.Info("exec copying")
		go io.Copy(out, eout)
		go io.Copy(out, sout)

	}

	glog.Info("exec starting ", p.Complete)
	err = eCmd.Start()
	if err != nil {
		glog.Error(err)
		out.Write([]byte(err.Error() + "\n\n"))
		p.Status = f.FAIL
		return
	}

	glog.Info("exec waiting")
	err = eCmd.Wait()

	glog.Info("exec cmd complete")

	if err != nil {
		glog.Error("command failed ", err)

		if msg, ok := err.(*exec.ExitError); ok {

			if status, ok := msg.Sys().(syscall.WaitStatus); ok {
				p.ExitStatus = status.ExitStatus()
				glog.Info("exit status: ", p.Status)
			}
		}
		// we prefer to return 0 for good or one for bad
		p.Status = f.FAIL
		return
	}

	p.Response = "exec command done"
	p.Status = f.SUCCESS

	glog.Info("executing command complete")
	return
}
