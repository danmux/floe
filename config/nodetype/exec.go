package nodetype

import "fmt"

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
	println("COMMAND >", cmd.(string))
	return 0, Opts{}, nil
}
