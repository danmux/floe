package nodetype

import (
	"fmt"
	"path/filepath"

	"github.com/floeit/floe/log"
)

type gitOpts struct {
	URL     string `json:"url"`
	SubDir  string `json:"sub-dir"`
	Ref     string `json:"ref"`      // what to checkout
	FromRef string `json:"from-ref"` // what to checkout and rebase onto Ref
}

// gitMerge is an executable node that checks out a hash and then
// checks out another - and then merges into it from the other
type gitMerge struct{}

func (g gitMerge) Match(ol, or Opts) bool {
	return true
}

func (g gitMerge) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {

	gop := gitOpts{}
	err := decode(in, &gop)
	if err != nil {
		return 255, nil, err
	}

	if gop.URL == "" {
		return 255, nil, fmt.Errorf("problem getting git url option")
	}
	if gop.Ref == "" {
		return 255, nil, fmt.Errorf("problem getting ref option")
	}
	if gop.FromRef == "" {
		return 255, nil, fmt.Errorf("problem getting from ref option")
	}

	output <- "git checkout: " + gop.URL + " merge into: " + gop.Ref + " from: " + gop.FromRef

	log.Debug("GIT merge ", gop.URL, " merge into: ", gop.Ref, " from: ", gop.FromRef)
	return 0, nil, nil
}

// gitCheckout checks out a has from a url
type gitCheckout struct{}

func (g gitCheckout) Match(ol, or Opts) bool {
	return true
}

func (g gitCheckout) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {
	gop := gitOpts{}
	err := decode(in, &gop)
	if err != nil {
		return 255, nil, err
	}
	if gop.Ref == "" {
		return 255, nil, fmt.Errorf("problem getting ref option")
	}
	if gop.URL == "" {
		return 255, nil, fmt.Errorf("problem getting git url option")
	}

	log.Debug("GIT clone ", gop.URL, "into:", gop.Ref, "into:", gop.SubDir)

	// for testing
	if gop.URL == "git@github.com:floeit/floe-test.git" {
		output <- "in dir: /Users/Dan/.flow/spaces/danmux/ws/h1-12/src/github.com/floeit"
		output <- "git clone --branch master --depth 1 git@github.com:floeit/floe-test.git"
		output <- "Cloning into 'floe'..."
		return 0, nil, nil
	}

	// git clone --branch mytag0.1 --depth 1 https://example.com/my/repo.git
	args := []string{"clone", "--branch", gop.Ref, "--depth", "1", gop.URL}
	status := doRun(filepath.Join(ws.BasePath, gop.SubDir), nil, output, "git", args...)

	return status, nil, nil
}
