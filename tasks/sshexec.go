package tasks

import (
	f "floe/workflow/flow"
	"github.com/golang/glog"
	"io"
)

type SSHExecTask struct {
	task   ExecTask
	remote string

	//echo "cd danmux/m-playbook; ansible-playbook v3_play.yml -i staging/inventory -l bricks -C" | ssh vstag /bin/bash
}

func (ft SSHExecTask) Type() string {
	return "ssh exec"
}

func MakeSSHExecTask(node, remotePAth, cmd, path string) SSHExecTask {

	sshCmd := "\"" + cmd + "\""

	if remotePAth != "" {
		sshCmd = "\"cd " + remotePAth + "; " + cmd + "\""
	}

	sshCmd = sshCmd + " | ssh " + node + " /bin/bash"

	return SSHExecTask{
		remote: node,
		task:   MakeExecTask("echo", sshCmd, path),
	}
}

func (ft SSHExecTask) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("executing ssh command")

	ft.task.Exec(t, p, out)
}
