package nodetype

import (
	"strings"
	"testing"
)

func TestExec(t *testing.T) {
	e := exec{}
	op := make(chan string)
	go func() {
		for l := range op {
			println(l)
		}
	}()

	e.Execute(&Workspace{}, Opts{
		"cmd": "echo foo",
	}, op)

	e.Execute(&Workspace{}, Opts{
		"shell": "export",
	}, op)

	close(op)
}

func TestCmdAndArgs(t *testing.T) {
	e := exec{
		Cmd: "foo bar",
	}
	cmd, arg := e.cmdAndArgs()
	if cmd != "foo" {
		t.Error("cmd should be foo")
	}
	if arg[0] != "bar" {
		t.Error("arg should be bar")
	}

	e = exec{
		Shell: "foo bar",
	}
	cmd, arg = e.cmdAndArgs()
	if cmd != "bash" {
		t.Error("cmd should be bash", cmd)
	}
	if arg[0] != "-c" || arg[1] != "foo bar" {
		t.Error("arg should be '-c' 'foo bar'", arg)
	}
}

func TestEnvVars(t *testing.T) {
	op := make(chan string)
	var out []string
	captured := make(chan bool)
	go func() {
		for l := range op {
			out = append(out, l)
		}
		captured <- true
	}()

	e := exec{}
	e.Execute(&Workspace{
		BasePath: ".",
	}, Opts{
		"shell": "export",
		"env":   []string{"DAN=fart"},
	}, op)

	close(op)

	<-captured

	expected := []string{`DAN="fart"`, `FLOEWS="."`}
	for _, x := range expected {
		found := false
		for _, l := range out {
			if strings.Contains(l, x) {
				found = true
				break
			}
		}
		if !found {
			t.Error("did not find env var:", x)
			for _, o := range out {
				println(o)
			}
		}
	}
}

func TestExpandEnvOpts(t *testing.T) {
	t.Parallel()
	env := []string{"OOF=${ws}/oof"}
	env = expandEnvOpts(env, "base/path")
	if env[0] != "OOF=base/path/oof" {
		t.Error("expand failed", env[0])
	}
}
