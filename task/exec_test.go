package task

import (
	"testing"
)

func TestExec(t *testing.T) {
	ExecFrame(t, MakeExecTask("pwd", "-L", ""), 3)
	ExecFrame(t, MakeExecTask("ls", "-lrt", ""), 4)
}
