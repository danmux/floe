package tasks

import (
	"encoding/json"
	"floe/log"
	"floe/tasks"
	f "floe/workflow/flow"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

// for marshaling the value
type GitHashes struct {
	RepoUrl string
	Hashes  map[string]string
}

type TriggerOnGitPush struct {
	repoUrl  string
	branch   string
	interval time.Duration // how often to check
}

func (ft *TriggerOnGitPush) Type() string {
	return "gitpush"
}

func MakeGitPushTrigger(repoUrl, branch string, interval time.Duration) *TriggerOnGitPush {
	if interval < 1 {
		interval = 1
	}

	if branch == "" {
		branch = "_all" // find first new branch
	}

	return &TriggerOnGitPush{
		repoUrl:  repoUrl,
		branch:   branch,
		interval: interval * time.Second,
	}
}

// params are passed in and mutated with results
func (ft *TriggerOnGitPush) Exec(t *f.TaskNode, p *f.Params, out *io.PipeWriter) {
	glog.Info("starting git pull trigger ", p.Complete, out)

	for {
		time.Sleep(ft.interval)

		if ft.ExecOnce(t, p, out) {
			return
		}
	}
}

func (ft *TriggerOnGitPush) ExecOnce(t *f.TaskNode, p *f.Params, out *io.PipeWriter) bool {
	// get task folder location
	hashesFile := p.Props[f.KEY_TRIGGERS] + "/" + t.Id() + ".state.json"

	// load in log if available
	prevHash := loadPrevHashes(hashesFile)

	// get log from url
	gitCommand := tasks.MakeExecTask("git", "ls-remote "+ft.repoUrl, "")

	outCommands, err := gitCommand.ExecCapture(t, p, out)

	if err == nil && len(outCommands) > 2 {

		glog.Info("got some git hashes ", len(outCommands))

		latestHash := &GitHashes{
			RepoUrl: ft.repoUrl,
		}

		parseGitResponse(outCommands, latestHash)

		glog.Info("got hashes", latestHash.Hashes)

		branch, hash, gotNew := gotDifferentHash(prevHash, latestHash, ft.branch)

		if gotNew {
			glog.Info("got some new hash for: ", branch)

			// update the old one with the single match - so we can trap other changes
			if prevHash.Hashes == nil {
				prevHash.Hashes = map[string]string{}
			}
			prevHash.Hashes[branch] = hash

			storeHashes(prevHash, hashesFile)

			p.Props["git-trigger-id"] = t.Id()
			p.Props["git-trigger-hash"] = hash
			p.Props["git-trigger-branch"] = branch

			out.Write([]byte("Triggering: " + t.Id() + "\n"))

			p.Status = f.SUCCESS
			p.Response = "trigger done"
			return true
		} else {
			glog.Info("no new hash found: ", branch)
		}
	}
	return false
}

func gotDifferentHash(oldH *GitHashes, newH *GitHashes, checkBranch string) (branch, hash string, matched bool) {
	glog.Info("checking branch", checkBranch)
	if checkBranch == "_all" {
		// find first match or new branch
		for br, newK := range newH.Hashes {
			oldK, ok := oldH.Hashes[br]
			// got this key in old list
			if ok {
				if newK != oldK {
					return br, newK, true // got a new one
				}
			} else { // did not have this in old so is new
				return br, newK, true // got a new one
			}
		}
		return "", "", false
	}

	n, nok := newH.Hashes[checkBranch]
	o, ook := oldH.Hashes[checkBranch]

	if nok {
		if ook { // got one in both - if hashes match then it is not new
			if n == o {
				return "", "", false
			}
		}
		// if we didnt have one in old or if they are different
		return n, checkBranch, true
	}

	return "", "", false
}

func parseGitResponse(lines []string, hashes *GitHashes) {
	// map the lines by branch
	glog.Info("parsing git list")

	hashes.Hashes = map[string]string{}
	for _, l := range lines[2:] { // from 2 onwards 1 = command 0 = empty
		sl := strings.Fields(l)

		if len(sl) > 1 {
			dp := strings.Split(sl[1], "/")
			if len(dp) > 2 {
				hashes.Hashes[dp[2]] = sl[0]
			} else if len(dp) == 1 {
				hashes.Hashes[dp[0]] = sl[0]
			}
		}
	}
}

func loadPrevHashes(hashFile string) *GitHashes {
	glog.Info("loading hashes from: ", hashFile)
	lastHashes := &GitHashes{}
	body, err := ioutil.ReadFile(hashFile)
	if err == nil {
		err = json.Unmarshal(body, lastHashes) // load it into myself
	}
	if err != nil {
		glog.Warning("hashes unmarshal error: ", err.Error())
	}
	return lastHashes
}

func storeHashes(hashes *GitHashes, hashFile string) {
	glog.Info("storing hashes to: ", hashFile)
	b, err := json.MarshalIndent(hashes, "", " ")
	if err != nil {
		glog.Warning("hashes marshall error: ", err.Error())
	}

	err = ioutil.WriteFile(hashFile, b, 0640)

	if err != nil {
		glog.Warning("hashes save error: ", err.Error())
	}
}
