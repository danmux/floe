package path

import (
	"os/user"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	t.Parallel()

	usr, _ := user.Current()
	hd := usr.HomeDir

	fix := []struct {
		in  string
		out string
		e   bool
	}{
		{in: "~/", out: "", e: true},      // too short
		{in: "~", out: "", e: true},       // too short
		{in: "~/test", out: hd + "/test"}, // sub ~
		{in: "/test/~", out: "/test/~"},   // dont ~
		{in: "test/foo", out: "test/foo"},
		{in: "/test/foo", out: "/test/foo"},
	}
	for i, f := range fix {
		ep, err := Expand(f.in)
		if (err == nil && f.e) || (err != nil && !f.e) {
			t.Errorf("test %d expected error mismatch", i)
		}
		if ep != f.out {
			t.Errorf("test %d failed, wanted: %s got: %s", i, f.out, ep)
		}
	}

	ep, _ := Expand("%tmp/test/bar")
	fPos := strings.Index(ep, "/floe")
	if fPos < 5 {
		t.Error("tmp expansion failed", ep)
	}
	if strings.Index(ep, "/test/bar") < fPos {
		t.Error("tmp expansion prefix... isn't ", ep)
	}
}
