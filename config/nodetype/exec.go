package nodetype

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/floeit/floe/exe"
	"github.com/floeit/floe/log"
)

// exec node executes an external task
type exec struct {
	Cmd    string
	Shell  string
	Args   []string
	SubDir string `json:"sub-dir"`
	Env    []string
}

func (e exec) Match(ol, or Opts) bool {
	return true
}

func (e exec) cmdAndArgs() (cmd string, args []string) {
	cmd = e.Cmd
	shell := false
	if cmd == "" {
		cmd = e.Shell
		shell = true
	}
	args = e.Args
	if len(args) == 00 {
		p := strings.Split(cmd, " ")
		if len(p) > 1 {
			cmd = p[0]
			args = p[1:]
		}
	}
	if shell {
		args = []string{"-c", fmt.Sprintf(`%s %s`, cmd, strings.Join(args, " "))}
		cmd = "bash"
	}
	return cmd, args
}

func (e exec) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {
	err := decode(in, &e)
	if err != nil {
		return 255, nil, err
	}

	cmd, args := e.cmdAndArgs()

	if cmd == "" {
		return 255, nil, fmt.Errorf("missing cmd or shell option")
	}
	// expand the workspace var
	e.Env = expandEnvOpts(e.Env, ws.BasePath)

	e.Env = append(e.Env, "FLOEWS="+ws.BasePath)

	status := doRun(filepath.Join(ws.BasePath, e.SubDir), e.Env, output, cmd, args...)

	return status, Opts{}, nil
}

func doRun(dir string, env []string, output chan string, cmd string, args ...string) int {
	stop := make(chan bool)
	out := make(chan string)

	output <- "in dir: " + dir + "\n"
	go func() {
		for o := range out {
			output <- o
		}
		stop <- true
	}()

	status := exe.Run(log.Log{}, out, env, dir, cmd, args...)

	// wait for output to complete
	<-stop

	if status != 0 {
		output <- fmt.Sprintf("\nexited with status: %d", status)
	}

	return status
}

// expand the workspace template item with the actual workspace
func expandEnvOpts(es []string, path string) []string {
	ne := make([]string, len(es))
	for i, e := range es {
		ne[i] = strings.Replace(e, "{{ws}}", path, -1)
	}
	return ne
}
