package task

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

type ExecTask struct {
	cmd    string
	args   string
	path   string // path relative to the workspace
	config TaskConfig
}

func (ft ExecTask) Type() string {
	return "execute"
}

func MakeExecTask(cmd, args, path string) ExecTask {
	return ExecTask{
		cmd:  cmd,
		args: args,
		path: path,
		config: TaskConfig{
			Command: fmt.Sprintf("exec: %v %v in %v", cmd, args, path),
		},
	}
}

func (ft ExecTask) Exec(ctx *Context, p *par.Params, out *io.PipeWriter) {
	log.Info("ExecTask.Exec")

	cmd, ok := p.Props["cmd"]
	// if no passed in cmd use defualt
	if !ok {
		cmd = ft.cmd
	}

	if cmd == "" {
		p.Status = par.StFail
		p.Response = "no cmd specified"
		return
	}

	args, ok := p.Props["args"]
	// if no passed in args use defualt
	if !ok {
		args = ft.args
	}

	log.Infof("Exec Cmd: <%s> Args: <%s> ", cmd, args)
	argStr := cmd + " " + args

	eCmd := exec.Command("bash", "-c", argStr)

	// this is mandatory
	eCmd.Dir = filepath.Join(ctx.WorkspacePath, ft.path)
	log.Info("In working directory: ", eCmd.Dir)

	var err error
	// out can be nil - it is only set for the first executing thread
	if out != nil {
		out.Write([]byte(ft.path + "$ " + argStr + "\n\n"))

		sout, err := eCmd.StdoutPipe()
		if err != nil {
			log.Info(err)
			p.Status = par.StFail
			return
		}
		eout, err := eCmd.StderrPipe()
		if err != nil {
			log.Error(err)
			p.Status = par.StFail
			return
		}

		log.Debug("Exec copying")
		go io.Copy(out, eout)
		go io.Copy(out, sout)

	}

	log.Debug("Exec starting ")
	err = eCmd.Start()
	if err != nil {
		log.Error(err)
		out.Write([]byte(err.Error() + "\n\n"))
		p.Status = par.StFail
		return
	}

	log.Debug("Exec waiting")
	err = eCmd.Wait()

	log.Debug("exec cmd complete")

	if err != nil {
		log.Error("Command failed ", err)

		if msg, ok := err.(*exec.ExitError); ok {

			if status, ok := msg.Sys().(syscall.WaitStatus); ok {
				p.ExitStatus = status.ExitStatus()
				log.Info("exit status: ", p.ExitStatus)
			}
		}
		// we prefer to return 0 for good or one for bad
		p.Status = par.StFail
		return
	}

	p.Response = "Exec ommand done"
	p.Status = par.StSuccess

	log.Info("Executing command complete")
	return
}

func (ft ExecTask) Config() TaskConfig {
	return ft.config
}

// ExecCapture execute the command but capture the output in string array
// forward = shall we forward to the command list (to show in the web page)
// most triggers which loop round - should set this false
func (ft ExecTask) ExecCapture(ctx *Context, p *par.Params, out *io.PipeWriter, forward bool) ([]string, error) {
	log.Info("exec capture", p.TaskID)
	var err error
	commandOutput := []string{}

	rp, wp := io.Pipe()

	// start the threads to monitor the reader
	go func() {
		scanner := bufio.NewScanner(rp)
		for scanner.Scan() {
			t := scanner.Text()
			log.Info("trigger exec out: ", t)
			commandOutput = append(commandOutput, t)
			if forward {
				out.Write([]byte(t + "\n")) // forward it on for display
			}
		}
		if err = scanner.Err(); err != nil {
			log.Error("There was an error with the scanner in exec capture", err)
		}
	}()

	// and add it to the results
	ft.Exec(ctx, p, wp)

	log.Info("Exec Captured: ", commandOutput)
	return commandOutput, err
}
