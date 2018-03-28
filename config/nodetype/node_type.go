package nodetype

// NType are the node types
type NType string

// NType reserved node types
const (
	NtEnd      NType = "end" // the special end node
	NtData     NType = "data"
	NtExec     NType = "exec"
	NtGitMerge NType = "git-merge"
)

// NodeType is the interface for an option comparing node
type NodeType interface {
	Match(Opts, Opts) bool
	Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error)
}

var nts = map[NType]NodeType{
	NtData:     data{},
	NtGitMerge: gitMerge{},
	NtExec:     exec{},
}

// GetNodeType returns the node from the given the type and opts
func GetNodeType(ty string) NodeType {
	return nts[NType(ty)]
}
