package config

import (
	"fmt"

	nt "github.com/floeit/floe/config/nodetype"
)

// FlowRef is a reference that uniquely identifies a flow
type FlowRef struct {
	ID  string
	Ver int
}

func (f FlowRef) String() string {
	if f.ID == "" {
		return "na"
	}
	return fmt.Sprintf("%s-%d", f.ID, f.Ver)
}

// NonZero returns true if this ref has been assigned nonzero values
func (f FlowRef) NonZero() bool {
	return f.ID != "" && f.Ver != 0
}

// Equal returns true if all fields in f anf g are equal
func (f FlowRef) Equal(g FlowRef) bool {
	return f.ID == g.ID && f.Ver == g.Ver
}

// Flow is a serialisable Flow Config, a definition of a flow. It is uniquely identified
// by an ID and Version.
type Flow struct {
	ID  string // url friendly ID - computed from the name if not given
	Ver int    // flow version, together with an ID form a global compound unique key

	Name         string   // human friendly name
	ReuseSpace   bool     `yaml:"reuse-space"`   // if true then will use the single workspace and will mutex with other instances of this Flow
	HostTags     []string `yaml:"host-tags"`     // tags that must match the tags on the host
	ResourceTags []string `yaml:"resource-tags"` // tags that if any flow is running with any matching tags then don't launch

	// The Various node types included in this flow
	Triggers []*node
	Tasks    []*node
}

func (f *Flow) matchTriggers(eType string, opts *nt.Opts) []*node {
	res := []*node{}
	for _, s := range f.Triggers {
		if s.matchedTriggers(eType, opts) {
			res = append(res, s)
		}
	}
	return res
}

// Node returns the node matching id
func (f *Flow) Node(id string) *node {
	for _, s := range f.Tasks {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (f *Flow) matchTag(tag string) []*node {
	res := []*node{}
	for _, s := range f.Tasks {
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

	ids := map[string]int{}
	for i, t := range f.Triggers {
		if err := t.zero(NcTrigger, fr); err != nil {
			return fmt.Errorf("%s %d - %v", NcTrigger, i, err)
		}
		ids[t.id()]++
	}
	for i, t := range f.Tasks {
		if err := t.zero(NcTask, fr); err != nil {
			return fmt.Errorf("%s %d - %v", NcTask, i, err)
		}
		ids[t.id()]++
	}

	// for _, class := range []NodeClass{NcMerge, NcTrigger, NcTask} {
	// 	nl := f.classToList(class)
	// 	ids := map[string]int{}
	// 	for i, t := range nl {
	// 		if err := t.zero(class, fr); err != nil {
	// 			return fmt.Errorf("%s %d - %v", class, i, err)
	// 		}
	// 		ids[t.id()]++
	// 	}
	for k, c := range ids {
		if c != 1 {
			return fmt.Errorf("%d nodes have id: %s", c, k)
		}
	}

	// convert to json-able options
	f.fixupOpts()

	return nil
}

func (f *Flow) fixupOpts() {
	for _, v := range f.Triggers {
		v.Opts.Fixup()
	}
	for _, v := range f.Tasks {
		v.Opts.Fixup()
	}
}
