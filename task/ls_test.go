package task

import (
	"testing"
)

func TestLs(t *testing.T) {
	p := ExecFrame(t, MakeLsTask("."), 0)
	t.Log(p.Props)

	_, ok := p.Props["testy_file.txt"]
	if !ok {
		t.Error("should have got test in props")
	}
}
