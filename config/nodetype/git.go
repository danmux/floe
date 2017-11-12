package nodetype

import (
	"fmt"

	"github.com/floeit/floe/log"
)

// gitMerge is an executable node that executes a git merge
type gitMerge struct{}

func (g gitMerge) Match(ol, or Opts) bool {
	return true
}

func (g gitMerge) Execute(ws Workspace, in Opts, output chan string) (int, Opts, error) {
	from, ok := in.string("from_hash")
	if !ok {
		return 255, nil, fmt.Errorf("problem getting from_hash string option")
	}
	to, ok := in.string("to_hash")
	if !ok {
		return 255, nil, fmt.Errorf("problem getting to_hash string option")
	}

	log.Debug("GIT merge command thing", from, to)
	return 0, nil, nil
}

func (g gitMerge) CastOpts(in *Opts) {}
