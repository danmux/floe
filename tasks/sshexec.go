package tasks

import (
	f "floe/workflow/flow"
	"fmt"
	"github.com/golang/glog"
	"io"
)

type SSHExecTask struct {
	task   ExecTask
	remote string
	config f.TaskConfig

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

func (ft SSHExecTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing ssh command")

	ft.task.Exec(t, p, out)
}

func (ft *SSHExecTask) makeCommand(node, remotePath, cmd string) {
	ft.config.Command = fmt.Sprintf("on node: %v in remote folder: %v execute %v", node, remotePath, cmd)
}

func (ft SSHExecTask) Config() f.TaskConfig {
	return ft.config
}
