package exe

import (
	"bufio"
	"io"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()

	var output []string
	var err error
	ended := false
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			t := scanner.Text()
			output = append(output, t)
		}
		ended = true
		err = scanner.Err()
	}()

	status := Run(&tLog{t: t}, "echo", "hello world", ".", pw)
	if status != 0 {
		t.Error("echo failed", status)
	}
	if !ended {
		t.Error("pipes not closed - maybe goroutine leak")
	}
	if err != nil {
		t.Error("scanner error", err)
	}
	if output[2] != "hello world" {
		t.Error("bad output", output)
	}

	// confirm bad command fails no command found
	status = Run(&tLog{t: t}, "echop", `hello world`, "", nil)
	if status != 1 {
		t.Error("status should have been 1", status)
	}
}

func TestRunOutput(t *testing.T) {
	t.Parallel()

	out, status := RunOutput(&tLog{t: t}, "echo", `hello world`, "")
	if status != 0 {
		t.Fatal("echo failed", status)
	}
	if out[2] != "hello world" {
		t.Errorf("bad output >%s<", out[2])
	}
}

type tLog struct {
	t *testing.T
}

func (l *tLog) Info(args ...interface{}) {
	l.t.Log("INFO", args)
}
func (l *tLog) Debug(args ...interface{}) {
	l.t.Log("DEBUG", args)
}
func (l *tLog) Error(args ...interface{}) {
	l.t.Log("ERROR", args)
}
func (l *tLog) Infof(format string, args ...interface{}) {
	l.t.Logf("INFO - "+format, args...)
}
