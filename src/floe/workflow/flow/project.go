package flow

import (
	// "fmt"
	// "sync"
	"encoding/json"
)

type Project struct {
	Name  string
	Flows map[string]*FlowLauncher
}

func MakeProject(name string) *Project {

	return &Project{
		Name:  name,
		Flows: map[string]*FlowLauncher{},
	}
}

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
