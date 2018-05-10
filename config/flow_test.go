package config

import "testing"

var flow = &Flow{
	Name: "flow-1",
	Triggers: []*node{
		&node{
			Name: "Start",
		},
	},
	Tasks: []*node{
		&node{
			Name:   "First Task",
			Listen: "trigger.good",
		},
		&node{
			Name:  "Merge Thing",
			Class: "merge",
			Wait:  []string{"task.first.good"},
		},
	},
}

func TestZero(t *testing.T) {

	err := flow.Zero()
	if err != nil {
		t.Error(err)
	}

	if flow.Triggers[0].ID != "start" {
		t.Error("zero failed", flow.Triggers[0].ID)
	}

	if flow.Tasks[0].ID != "first-task" {
		t.Error("zero failed", flow.Tasks[0].ID)
	}
	if flow.Tasks[0].Class != NcTask {
		t.Error("zero failed", flow.Tasks[0].Class)
	}
	if flow.Tasks[1].Class != NcMerge {
		t.Error("zero failed on merge node", flow.Tasks[1].Class)
	}
}

func TestMatchTag(t *testing.T) {

	err := flow.Zero()
	if err != nil {
		t.Error(err)
	}

	matches := flow.MatchTag("trigger.good")
	if len(matches) != 1 {
		t.Error("did not find task node")
	}

	matches = flow.MatchTag("task.first.good")
	if len(matches) != 1 {
		t.Error("did not find merge node")
	}
}
