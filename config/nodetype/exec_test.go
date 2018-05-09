package nodetype

import (
	"io/ioutil"
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
	opts := Opts{
		"shell": "export",
		"env":   []string{"DAN=fart"},
	}
	testNode(t, "exe env vars", exec{}, opts, []string{`DAN="fart"`, `FLOEWS="."`})
}

func testNode(t *testing.T, msg string, nt NodeType, opts Opts, expected []string) bool {
	op := make(chan string)
	var out []string
	captured := make(chan bool)
	go func() {
		for l := range op {
			out = append(out, l)
		}
		captured <- true
	}()

	tmp, err := ioutil.TempDir("", "floe-test")
	if err != nil {
		t.Fatal("can't create tmp dir")
	}

	nt.Execute(&Workspace{
		BasePath:   ".",
		FetchCache: tmp,
	}, opts, op)

	close(op)

	<-captured

	prob := false
	for _, x := range expected {
		found := false
		for _, l := range out {
			if strings.Contains(l, x) {
				found = true
				break
			}
		}
		if !found {
			prob = true
			t.Error(msg, "did not find expected:", x)
		}
	}
	// output the output if there was a problem
	if prob {
		t.Log("cache is at:", tmp)
		for _, o := range out {
			t.Log(o)
		}
	}
	return prob
}

func TestExpandEnvOpts(t *testing.T) {
	t.Parallel()
	env := []string{"OOF=${ws}/oof"}
	env = expandEnvOpts(env, "base/path")
	if env[0] != "OOF=base/path/oof" {
		t.Error("expand failed", env[0])
	}
}
