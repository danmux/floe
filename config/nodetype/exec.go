package nodetype

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/floeit/floe/exe"
	"github.com/floeit/floe/log"
)

// exec node executes an external task
type exec struct{}

func (e exec) Match(ol, or Opts) bool {
	return true
}

func (e exec) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {
	// TODO - consider mapstructure
	cmd := ""
	if c, ok := in["cmd"]; ok {
		cmd = c.(string)
	} else {
		return 255, nil, fmt.Errorf("missing cmd option")
	}

	args := ""
	if a, ok := in["args"]; ok {
		args = a.(string)
	}
	// if no explicit args then look at the cmd if it is in the form of "arg cmd.."
	if args == "" {
		p := strings.Split(cmd, " ")
		if len(p) > 1 {
			args = cmd[len(p[0]):]
			cmd = p[0]
		}
	}

	pr, pw := io.Pipe()
	stop := make(chan bool, 1)

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			output <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			output <- "scanning output failed with: " + err.Error()
		}
		close(stop)
	}()

	status := exe.Run(log.Log{}, cmd, args, ws.BasePath, pw)

	// wait for scanner to complete
	<-stop

	return status, Opts{}, nil
}
