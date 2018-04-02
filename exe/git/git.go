package git

import (
	"strings"

	"github.com/floeit/floe/exe"
)

type logger interface {
	Info(...interface{})
	Debug(...interface{})
	Error(...interface{})
	Infof(format string, args ...interface{})
}

type GitHashes struct {
	RepoUrl string
	Hashes  map[string]string
}

func GitLs(log logger, url string) (*GitHashes, bool) {

	gitOut, status := exe.RunOutput(log, "git", "ls-remote "+url, "")
	if status != 0 {
		return nil, false
	}
	latestHash := &GitHashes{
		RepoUrl: url,
	}

	parseGitResponse(gitOut, latestHash)
	return latestHash, true
}

func parseGitResponse(lines []string, hashes *GitHashes) {
	// map the lines by branch
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
