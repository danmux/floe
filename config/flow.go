package config

import (
	"fmt"

	nt "github.com/floeit/floe/config/nodetype"
)

// FlowRef uniquely identifies a flow
type FlowRef struct {
	ID  string
	Ver int
}

func (f FlowRef) String() string {
	return fmt.Sprintf("%s-%d", f.ID, f.Ver)
}

// Flow is a serialisable Flow Config
type Flow struct {
	Name       string   // human friendly name
	ID         string   // url friendly ID - computed from the name if not given
	Ver        int      // flow version
	ReuseSpace bool     `yaml:"reuse-space"` // if true then will use the single workspace and will mutex with other instances of this Flow
	HostTags   []string `yaml:"host-tags"`   // tags that must match the tags on the host

	// the Various node types
	Subs   []*task
	Tasks  []*task
	Pubs   []*task
	Merges []*task
}

func (f *Flow) matchSubs(eType string, opts *nt.Opts) []*task {
	res := []*task{}
	for _, s := range f.Subs {
		if s.matchedSub(eType, opts) {
			res = append(res, s)
		}
	}
	return res
}

func (f *Flow) matchTag(class NodeClass, tag string) []*task {
	res := []*task{}
	nl := f.classToList(class)
	for _, s := range nl {
		if s.matched(tag) {
			res = append(res, s)
		}
	}
	return res
}

func (f *Flow) setName(n string) {
	f.Name = n
}
func (f *Flow) setID(i string) {
	f.ID = i
}
func (f *Flow) name() string {
	return f.Name
}
func (f *Flow) id() string {
	return f.ID
}

func (f *Flow) zero() error {
	if err := zeroNID(f); err != nil {
		return err
	}

	fr := FlowRef{
		ID:  f.ID,
		Ver: f.Ver,
	}

	for _, class := range []NodeClass{NcMerge, NcPub, NcSub, NcTask} {
		nl := f.classToList(class)
		for i, t := range nl {
			if err := t.zero(class, fr); err != nil {
				return fmt.Errorf("%s %d - %v", class, i, err)
			}
		}
	}

	return nil
}

func (f *Flow) classToList(class NodeClass) []*task {
	nl := []*task{}
	switch class {
	case NcMerge:
		nl = f.Merges
	case NcPub:
		nl = f.Pubs
	case NcSub:
		nl = f.Subs
	case NcTask:
		nl = f.Tasks
	}
	return nl
}
