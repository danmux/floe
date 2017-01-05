package nodetype

import (
	"fmt"
	"log"
)

// git push subs node
type gitPush struct{}

func (g gitPush) Match(ol, or Opts) bool {
	return ol.cmpString("url", or)
}

func (g gitPush) Execute(in Opts) (int, Opts, error) {
	return 0, nil, nil
}

// gitMerge is an executable node that merges in a
type gitMerge struct{}

func (g gitMerge) Match(ol, or Opts) bool {
	return true
}

func (g gitMerge) Execute(in Opts) (int, Opts, error) {
	from, ok := in.string("from_hash")
	if !ok {
		return 255, nil, fmt.Errorf("problem getting from_hash string option")
	}
	to, ok := in.string("to_hash")
	if !ok {
		return 255, nil, fmt.Errorf("problem getting to_hash string option")
	}

	log.Println("GIT merge command thing", from, to)
	return 0, nil, nil
}
