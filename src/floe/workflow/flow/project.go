package flow

import (
	// "fmt"
	// "sync"
	"encoding/json"
)

// the top level data structure that drives everytihng - contains many flows
// TODO - construct a rule structure that allows a succesfull workflow completion to trigger another workflow
type Project struct {
	Name        string
	Flows       map[string]*FlowLauncher
	LastResults map[string]*FlowLaunchResult // a set of response stats by task id in our workflow for the last run
}

func MakeProject(name string) *Project {

	return &Project{
		Name:        name,
		Flows:       map[string]*FlowLauncher{},
		LastResults: map[string]*FlowLaunchResult{},
	}
}

func (p *Project) AddFlow(f *FlowLauncher) {
	p.Flows[MakeID(f.Name)] = f
}

func (p *Project) ColectResults() {
	for fid, flo := range p.Flows {
		if flo.LastRunResult != nil {
			flo.LastRunResult.FlowId = fid
		}
		p.LastResults[fid] = flo.LastRunResult
	}
}

// a project structure is just a list of flowstructs so we can render the project graph
type ProjectStruct struct {
	Flows []FlowStruct
}

func (p Project) ToJson() []byte {
	ps := ProjectStruct{
		Flows: []FlowStruct{},
	}
	for _, f := range p.Flows {
		ps.Flows = append(ps.Flows, f.GetStructure())
	}

	sJson, err := json.MarshalIndent(&ps, "", "  ")
	if err != nil {
		return nil
	}
	return sJson
}
