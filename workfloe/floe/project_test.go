package floe

import (
	"encoding/json"
	"testing"
)

func TestProject(t *testing.T) {
	p := ProjectOverview{}
	b, _ := json.Marshal(p)
	println(string(b))
}
