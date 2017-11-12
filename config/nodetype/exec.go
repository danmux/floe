package nodetype

import (
	"fmt"
	"log"
	"time"
)

// exec node executes an external task
type exec struct{}

func (e exec) Match(ol, or Opts) bool {
	return true
}

func (e exec) Execute(ws Workspace, in Opts, output chan string) (int, Opts, error) {
	cmd, ok := in["cmd"]
	if !ok {
		return 255, nil, fmt.Errorf("missing cmd option")
	}

	log.Println("COMMAND >", cmd.(string)) // TODO - it
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second * 1)
		output <- fmt.Sprintf("something after %d seconds", i)
	}
	log.Println("COMMAND >", cmd.(string), "DONE")

	return 0, Opts{}, nil
}

func (e exec) CastOpts(in *Opts) {}
