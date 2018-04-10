package nodetype

import "github.com/mitchellh/mapstructure"

// NType are the node types
type NType string

// NType reserved node types
const (
	NtEnd         NType = "end" // the special end node
	NtData        NType = "data"
	NtTimer       NType = "timer"
	NtExec        NType = "exec"
	NtGitMerge    NType = "git-merge"
	NtGitCheckout NType = "git-checkout"
)

// NodeType is the interface for a node. All implementations on NodeType are stateless
// THe Execute method must be a pure(ish) function operating on in and returning an out Opts
type NodeType interface {
	Match(Opts, Opts) bool
	Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error)
}

var nts = map[NType]NodeType{
	NtData:        data{},
	NtTimer:       timer{},
	NtExec:        exec{},
	NtGitMerge:    gitMerge{},
	NtGitCheckout: gitCheckout{},
}

// GetNodeType returns the node from the given the type and opts
func GetNodeType(ty string) NodeType {
	return nts[NType(ty)]
}

func decode(input interface{}, output interface{}) error {

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   output,
		TagName:  "json",
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}
