package flow

// import (
// 	"errors"
// 	"fmt"
// 	"strings"
// )

// type Params struct {
// 	FlowName string // these three make up a unique ID for the task
// 	ThreadId int
// 	TaskName string

// 	TaskType string
// 	Status   int
// 	Response string
// 	Props    Props
// }

type Results struct {
	Fl      *FlowLauncher
	Results map[string][]*Params
}

func MakeResults(fl *FlowLauncher) Results {
	return Results{
		Fl:      fl,
		Results: map[string][]*Params{},
	}
}
