package nodetype

import (
	"fmt"
	"log"
)

// data
type data struct{}

func (d data) Match(qs, as Opts) bool {
	return true
}

func (d data) Execute(ws Workspace, in Opts, output chan string) (int, Opts, error) {
	cmd, ok := in["cmd"]
	if !ok {
		return 255, nil, fmt.Errorf("missing cmd option")
	}

	log.Println("COMMAND >", cmd.(string)) // TODO - it

	return 0, Opts{}, nil
}

func (d data) CastOpts(in *Opts) {}
