package nodetype

import (
	"fmt"
	"log"
)

// git node
type exec struct{}

func (e exec) Match(ol, or Opts) bool {
	return true
}

func (e exec) Execute(in Opts) (int, Opts, error) {
	cmd, ok := in["cmd"]
	if !ok {
		return 255, nil, fmt.Errorf("missing cmd option")
	}
	log.Println("COMMAND >", cmd.(string))
	// time.Sleep(time.Millisecond * time.Duration(500+rand.Intn(300)))
	// time.Sleep(time.Second)
	return 0, Opts{}, nil
}
