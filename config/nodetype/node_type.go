package nodetype

// NodeType is the interface for an option comparing node
type NodeType interface {
	Match(Opts, Opts) bool
	Execute(ws Workspace, in Opts, output chan string) (int, Opts, error)
	CastOpts(in *Opts)
}

var nts = map[string]NodeType{
	"data":      data{},
	"git-merge": gitMerge{},
	"exec":      exec{},
}

// GetNodeType returns the node from the given the type and opts
func GetNodeType(nType string) NodeType {
	return nts[nType]
}
