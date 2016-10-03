package agent

import (
	// "encoding/json"
	"testing"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/testfloe"
)

func TestAgentExec(t *testing.T) {
	log.SetLevel(8)

	a := NewAgent("a1", "test agent 1")
	a.Setup("test", testfloe.GetFloes, "~/tmp/floe")

	// a.Exec("test-build", time.Millisecond)

	// a.Exec("test-build", time.Millisecond)

	// a.Exec("test-build", time.Millisecond)

	// a.Exec("test-build", time.Millisecond)

	time.Sleep(10 * time.Second)

	// r := a.project.Runs("test-build")
	// b, _ := json.MarshalIndent(r, "", "  ")
	// t.Log(string(b))
}
