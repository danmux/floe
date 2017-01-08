package nodetype

// Opts are the options on the node type that will be compared to those on the event
type Opts map[string]interface{}

func (o Opts) string(key string) (string, bool) {
	si, ok := o[key]
	if !ok {
		return "", false
	}
	s, ok := si.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func (o Opts) cmpString(key string, or Opts) bool {
	sl, ok := o.string(key)
	if !ok {
		return false
	}
	sr, ok := or.string(key)
	if !ok {
		return false
	}
	return sl == sr
}

// MergeOpts merges l and r into a new Opts struct
func MergeOpts(l, r Opts) Opts {
	o := Opts{}
	for k, v := range l {
		o[k] = v
	}
	for k, v := range r {
		o[k] = v
	}
	return o
}

type Workspace struct {
	BasePath string
}

// NodeType is the interface for an option comparing node
type NodeType interface {
	Match(Opts, Opts) bool
	Execute(ws Workspace, in Opts) (int, Opts, error)
}

// GetNodeType returns the node from the given the type and opts
func GetNodeType(nType string) NodeType {
	switch nType {
	case "git-push":
		return gitPush{}
	case "git-merge":
		return gitMerge{}
	case "exec":
		return exec{}
	}
	return nil
}
