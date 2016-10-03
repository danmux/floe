package task

import (
	"fmt"
	"io"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

type SSHExecTask struct {
	task   ExecTask
	remote string
	config TaskConfig

	//echo "cd danmux/m-playbook; ansible-playbook v3_play.yml -i staging/inventory -l bricks -C" | ssh vstag /bin/bash
}

func (ft SSHExecTask) Type() string {
	return "ssh exec"
}

func MakeSSHExecTask(node, remotePath, cmd, path string) SSHExecTask {

	sshCmd := "\"" + cmd + "\""

	if remotePath != "" {
		sshCmd = "\"cd " + remotePath + "; " + cmd + "\""
	}

	sshCmd = sshCmd + " | ssh " + node + " /bin/bash"

	t := SSHExecTask{
		remote: node,
		task:   MakeExecTask("echo", sshCmd, path),
	}

	t.makeCommand(node, remotePath, cmd)

	return t
}

func (ft SSHExecTask) Exec(ctx *Context, p *par.Params, out *io.PipeWriter) {
	log.Info("executing ssh command")

	ft.task.Exec(ctx, p, out)
}

func (ft *SSHExecTask) makeCommand(node, remotePath, cmd string) {
	ft.config.Command = fmt.Sprintf("On node: %v in remote folder: %v execute %v", node, remotePath, cmd)
}

func (ft SSHExecTask) Config() TaskConfig {
	return ft.config
}
