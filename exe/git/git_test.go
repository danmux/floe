// +build integration_test

package git

import "testing"

func TestGitLs(t *testing.T) {
	output, ok := GitLs(&tLog{t: t}, "git@github.com:floeit/floe.git")
	if !ok {
		t.Error("git ls failed")
	}
	if len(output.Hashes) == 0 {
		t.Error("got no hashes")
	}
	for k, v := range output.Hashes {
		t.Log(k, v)
	}
}

type tLog struct {
	t *testing.T
}

func (l *tLog) Info(args ...interface{}) {
	l.t.Log("INFO", args)
}
func (l *tLog) Debug(args ...interface{}) {
	l.t.Log("DEBUG", args)
}
func (l *tLog) Error(args ...interface{}) {
	l.t.Log("ERROR", args)
}
func (l *tLog) Infof(format string, args ...interface{}) {
	l.t.Logf("INFO - "+format, args...)
}
