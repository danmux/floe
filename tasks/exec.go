package tasks

import (
	f "floe/workflow/flow"
	"io"
	"os/exec"
	// "strings"
	"bufio"
	"floe/log"
	"syscall"
)

type ExecTask struct {
	cmd  string
	args string
	path string // path relative to the workspace
}

func (ft ExecTask) Type() string {
	return "execute"
}

func MakeExecTask(cmd, args, path string) ExecTask {
	return ExecTask{
		cmd:  cmd,
		args: args,
		path: path,
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
	eCmd.Dir = t.WorkFlow().Params.Props[f.KEY_WORKSPACE] + ft.path
	glog.Info("working directory: ", eCmd.Dir)

	var err error
	// out can be nil - it is only set for the first executing thread
	if out != nil {
		out.Write([]byte(eCmd.Dir + "$ " + argstr + "\n\n"))

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

// execute the command but capture the output in string array
func (ft ExecTask) ExecCapture(t *f.TaskNode, p *f.Params, out *io.PipeWriter) ([]string, error) {
	glog.Info("exec capture", t.Id())
	var err error
	commandOutput := []string{}

	rp, wp := io.Pipe()

	// start the threads to monitor the reader
	go func() {
		scanner := bufio.NewScanner(rp)
		for scanner.Scan() {
			t := scanner.Text()
			glog.Info("trigger exec out: ", t)
			commandOutput = append(commandOutput, t)
			out.Write([]byte(t + "\n")) // forward it on for display
		}
		if err = scanner.Err(); err != nil {
			glog.Error("There was an error with the scanner in exec capture", err)
		}
	}()

	// and add it to the results
	ft.Exec(t, p, wp)

	glog.Info("Exec Captured: ", commandOutput)
	return commandOutput, err
}
