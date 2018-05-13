package nodetype

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/floeit/floe/exe"
	"github.com/floeit/floe/log"
)

const (
	shortRel = "./"
	wsSub    = "{{ws}}"
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

	// expand the workspace var and any env vars for the vars, command and args
	e.Env = expandEnvOpts(e.Env, ws.BasePath)
	for i, arg := range args {
		args[i] = expandEnv(arg, ws.BasePath)
	}
	cmd = expandWs(cmd, ws.BasePath)
	// use any cmd on the new env path, rather than current path
	cmd = useEnvPathCmd(cmd, e.Env)

	// add in the env var path to the workspace so scripts can use it
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
		ne[i] = expandEnv(e, path)
	}
	return ne
}

func expandEnv(e string, path string) string {
	parts := strings.Split(e, "=")
	if len(parts) == 2 {
		return parts[0] + "=" + expandEnvWsDot(parts[1], path)
	}
	return expandEnvWsDot(e, path)
}

func expandEnvWsDot(e string, path string) string {
	if len(e) == 0 {
		return e
	}
	// relative values substitution
	// any "." on its own or anything starting ./ but not ./..
	if len(e) == 1 && e[0] == '.' {
		e = wsSub
	} else if strings.HasPrefix(e, shortRel) && !strings.HasPrefix(e, "./...") {
		fmt.Printf("replacing in <%s>\n", e)
		e = strings.Replace(e, shortRel, wsSub+"/", 1)
	}

	return expandWs(e, path)
}

func expandWs(e string, path string) string {
	// avoid any //{{ws}}
	e = strings.Replace(e, "/"+wsSub, wsSub, -1)
	return os.ExpandEnv(strings.Replace(e, wsSub, path, -1))
}

func useEnvPathCmd(cmd string, env []string) string {
	// find path
	for _, e := range env {
		parts := strings.Split(e, "=")
		if len(parts) == 2 && parts[0] == "PATH" {
			c := lookPath(cmd, parts[1])
			if c != "" {
				return c
			}
			return cmd
		}
	}
	return cmd
}

// ErrNotFound is the error resulting if a path search failed to find an executable file.
var ErrNotFound = errors.New("executable file not found in $PATH")

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}

func lookPath(file string, path string) string {
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return file
		}
		return ""
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(path); err == nil {
			return path
		}
	}
	return ""
}
