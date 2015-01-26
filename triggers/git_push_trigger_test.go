package tasks

import (
	"bufio"
	"encoding/json"
	f "floe/workflow/flow"
	"fmt"
	"io"
	"testing"
)

func Test_LoadEmptyPushTrigger(t *testing.T) {
	prev := loadPrevHashes("empty.stat.json")
	if len(prev.Hashes) != 0 {
		t.Error("empty hash should be empty")
	}

	newH := &GitHashes{}
	branch, hash, ok := gotDifferentHash(prev, newH, "_all")

	if ok {
		t.Error("should not find stuff in empty hashes")

		if hash == "" {
			t.Error("cant have empty hash if hash found")
		}

		if branch == "" {
			t.Error("cant have empty branch if hash found")
		}
	}

	// got new hashes so must be different
	newH = &GitHashes{
		Hashes: map[string]string{
			"HEAD":   "70419e05b05b9d1eea9a01d530492f0c14457a93",
			"master": "70419e05b05b9d1eea9a01d530492f0c14457a93",
		},
	}

	_, hash, ok = gotDifferentHash(prev, newH, "_all")
	if !ok {
		t.Error("brand new hashes should trigger new")
	}

	if hash != newH.Hashes["HEAD"] {
		t.Error("brand new hashes triggered wrong hash", hash)
	}
}

func Test_SavePushTrigger(t *testing.T) {
	newH := &GitHashes{
		RepoUrl: "rep-url",
		Hashes: map[string]string{
			"branch1": "hash1",
			"branch2": "hash2",
		},
	}

	fn := "/tmp/test.stat.json"

	storeHashes(newH, fn)

	oldH := loadPrevHashes(fn)

	b, _ := json.MarshalIndent(newH, "", "  ")
	fmt.Println(string(b))

	b, _ = json.MarshalIndent(oldH, "", "  ")
	fmt.Println(string(b))

	if oldH.RepoUrl != newH.RepoUrl {
		t.Error("should have loaded old hashes url")
	}

	h1 := oldH.Hashes["branch1"]
	if h1 != newH.Hashes["branch1"] {
		t.Error("should have loaded old hash for b1")
	}

	h2 := oldH.Hashes["branch2"]
	if h2 != newH.Hashes["branch2"] {
		t.Error("should have loaded old hash for b2")
	}

	_, _, ok := gotDifferentHash(oldH, newH, "_all")
	if ok {
		t.Error("should have not found difference")
	}
}

func Test_SaveFirstOfAll(t *testing.T) {
	oldH := &GitHashes{
		RepoUrl: "rep-url",
		Hashes: map[string]string{
			"fbranch1": "fhash1",
			"fbranch2": "fhash2",
		},
	}

	newH := &GitHashes{
		RepoUrl: "rep-url",
		Hashes: map[string]string{
			"fbranch1": "fhash1",
			"fbranch2": "fhash2-new",
		},
	}

	branch, hash, ok := gotDifferentHash(oldH, newH, "_all")

	if !ok {
		t.Error("one hash was different but not spotted")
	}

	if hash != "fhash2-new" {
		t.Error("didnt find different hash", hash)
	}

	if branch != "fbranch2" {
		t.Error("didnt find different hash branch")
	}
}

func Test_ParseGitResponseTrigger(t *testing.T) {

	hashes := &GitHashes{}

	// N.B. dont loose the tab character in the following
	lines := []string{
		"command",
		"d79349c5da77fabf4d18f62d7e1e5abfd97d2382	HEAD",
		"f0da908ebb171873f4f5f1b557287a176ead88a0	refs/heads/PM-8466-DDBAC-Adapter",
		"78973361cd4150dea1495e5bb441bfe61d6e877a	refs/heads/PM-8469-Adapter-Router",
		"d79349c5da77fabf4d18f62d7e1e5abfd97d2382	refs/heads/development",
		"be9453d99ce1925c0649a7f8c004aaf140cdb8ee	refs/heads/master",
	}

	parseGitResponse(lines, hashes)

	if hashes.Hashes["PM-8469-Adapter-Router"] != "78973361cd4150dea1495e5bb441bfe61d6e877a" {
		t.Error("Parsing strings failed")
	}

	// for k, v := range hashes.Hashes {
	// 	fmt.Println(k, v)
	// }
}

func Test_RealGitPushTrigger(t *testing.T) {

	fmt.Println("fdghjkgfds000000000000")

	pt := MakeGitPushTrigger("git@github.com:floeit/floeit.github.io.git", "_all", 2)

	w := f.MakeWorkflow()

	tn := w.MakeTriggerNode("test-git-push-trigger", pt)

	p := &f.Params{
		Props: map[string]string{
			f.KEY_TRIGGERS:  "/tmp",
			f.KEY_WORKSPACE: "/tmp",
		},
	}

	w.Params = p

	rp, wp := io.Pipe()

	fmt.Println("fdghjkgfds")

	go func() {
		scanner := bufio.NewScanner(rp)
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Println(t)
		}
	}()

	pt.Exec(tn, p, wp)

}
