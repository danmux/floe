package nodetype

import (
	"fmt"
	"log"
)

// exec node executes an external task
type exec struct{}

func (e exec) Match(ol, or Opts) bool {
	return true
}

func (e exec) Execute(ws Workspace, in Opts) (int, Opts, error) {
	cmd, ok := in["cmd"]
	if !ok {
		return 255, nil, fmt.Errorf("missing cmd option")
	}

	log.Println("COMMAND >", cmd.(string)) // TODO - it

	return 0, Opts{}, nil
}
