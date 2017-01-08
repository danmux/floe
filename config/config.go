package config

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	nt "github.com/floeit/floe/config/nodetype"
)

// idFromName makes a file and URL/HTML friendly ID from the name.
func idFromName(name string) string {
	s := strings.Split(strings.ToLower(strings.TrimSpace(name)), " ")
	ns := strings.Join(s, "-")
	s = strings.Split(ns, ".")
	return strings.Join(s, "-")
}

func nameFromID(id string) string {
	s := strings.Split(strings.ToLower(strings.TrimSpace(id)), "-")
	return strings.Join(s, " ")
}

// Node is the thing that an event triggers some behavior in
type nid interface {
	setID(string)
	setName(string)
	name() string
	id() string
}

type Node interface {
	FlowRef() FlowRef
	NodeRef() NodeRef
	Class() NodeClass
	Execute(nt.Workspace, nt.Opts) (int, nt.Opts, error)
	IsStatusGood(int) bool
	TypeOfNode() string
	Waits() int
}

// NodeClass the type def for the types a Node can be
type NodeClass string

// NodeClass values
const (
	NcTask  NodeClass = "task"
	NcMerge NodeClass = "merge"
	NcSub   NodeClass = "sub"
	NcPub   NodeClass = "pub"
)

// NodeRef uniquely identifies a Node across time (versions)
type NodeRef struct {
	Class NodeClass
	ID    string
}

// trim trailing spaces and dots and hyphens
func trimNIDs(s string) string {
	return strings.Trim(s, " .-")
}

func zeroNID(n nid) error {
	name := trimNIDs(n.name())
	id := trimNIDs(n.id())

	if name == "" && id == "" {
		return errors.New("task id and name can not both be empty")
	}
	if id == "" {
		id = idFromName(name)
	}
	if strings.IndexAny(id, " .") >= 0 {
		return errors.New("a specified id can not contain spaces or full stops")
	}
	if name == "" {
		name = nameFromID(id)
	}

	n.setID(id)
	n.setName(name)
	return nil
}

type task struct {
	// what flow is this node attached to
	flowRef    FlowRef
	class      NodeClass
	Ref        NodeRef
	ID         string
	Name       string
	Listen     string
	Wait       []string // if used as a merge node this is an array of event tags to wait for
	Type       string
	Good       []int
	IgnoreFail bool
	Opts       nt.Opts // static config options
}

func (t *task) Execute(ws nt.Workspace, opts nt.Opts) (int, nt.Opts, error) {
	n := nt.GetNodeType(t.Type)
	if n == nil {
		return 255, nil, fmt.Errorf("no node type found: %s", t.Type)
	}
	inOpts := nt.MergeOpts(t.Opts, opts)
	return n.Execute(ws, inOpts)
}

func (t *task) IsStatusGood(status int) bool {
	// always return good if ignorefail
	if t.IgnoreFail {
		return true
	}
	// nothing specified assume 0 = good
	if len(t.Good) == 0 {
		return status == 0
	}
	// otherwise true if in specific list
	for _, s := range t.Good {
		if s == status {
			return true
		}
	}
	return false
}

func (t *task) FlowRef() FlowRef {
	return t.flowRef
}

func (t *task) NodeRef() NodeRef {
	return t.Ref
}

func (t *task) Class() NodeClass {
	return t.class
}

func (t *task) TypeOfNode() string {
	return t.Type
}

func (t *task) Waits() int {
	return len(t.Wait)
}

func (t *task) matched(eType string, opts *nt.Opts) bool {
	if opts != nil {
		if t.Type != eType {
			return false
		}
		n := nt.GetNodeType(eType)
		if n == nil {
			return false
		}
		// compare config options with the event options
		return n.Match(t.Opts, *opts)
	}
	// otherwise for all other tasks match on the Listen
	if t.Listen != "" && t.Listen == eType {
		return true
	}
	// or if any tags in the the Wait list match (merge nodes only)
	for _, tag := range t.Wait {
		if tag == eType {
			return true
		}
	}
	return false
}

func (t *task) setName(n string) {
	t.Name = n
}
func (t *task) setID(i string) {
	t.ID = i
}
func (t *task) name() string {
	return t.Name
}
func (t *task) id() string {
	return t.ID
}

func (t *task) zero(class NodeClass, flow FlowRef) error {
	if err := zeroNID(t); err != nil {
		return err
	}
	t.Ref = NodeRef{
		Class: class,
		ID:    t.ID,
	}
	t.flowRef = flow
	t.class = class
	return nil
}

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
	Name         string   // human friendly name
	ID           string   // url friendly ID - computed from the name if not given
	Ver          int      // flow version
	ReuseSpace   bool     `yaml:"reuse-space"`   // if true then will use the single workspace and will mutex with other instances of this Flow
	ResourceTags []string `yaml:"resource-tags"` // tags to group resources this flow needs
	HostTags     []string `yaml:"host-tags"`     // tags that must match the tags on the host

	// the Various node types
	Subs   []*task
	Tasks  []*task
	Pubs   []*task
	Merges []*task
}

func (f *Flow) match(class NodeClass, eType string, opts *nt.Opts) []*task {
	res := []*task{}
	nl := f.classToList(class)
	for _, s := range nl {
		if s.matched(eType, opts) {
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

// Config is the set of nodes and rules
type Config struct {
	Flows []*Flow
}

// FoundFlow is a struct returned from FindFlowsBySubs
// and can be used to decide on the best host to use
type FoundFlow struct {
	Ref          FlowRef
	ReuseSpace   bool
	ResourceTags []string
	HostTags     []string

	Nodes []Node
}

// FindFlowsBySubs finds all flows where its subs match the given params
func (c *Config) FindFlowsBySubs(eType string, opts nt.Opts) map[FlowRef]FoundFlow {
	res := map[FlowRef]FoundFlow{}
	for _, f := range c.Flows {
		ns := f.match(NcSub, eType, &opts)
		// found some matching nodes for this flow
		if len(ns) > 0 {
			// make sure this flow is in the results
			fr := ns[0].FlowRef()
			ff, ok := res[fr]
			if !ok {
				ff = FoundFlow{
					Ref:          fr,
					ReuseSpace:   f.ReuseSpace,
					ResourceTags: f.ResourceTags,
					HostTags:     f.HostTags,
				}
			}
			ff.Nodes = []Node{}
			for _, n := range ns {
				ff.Nodes = append(ff.Nodes, Node(n))
			}
			res[fr] = ff
		}
	}
	return res
}

// FindFlow finds the specific flow where its subs match the given params
func (c *Config) FindFlow(f FlowRef, eType string, opts nt.Opts) (FoundFlow, bool) {
	found := c.FindFlowsBySubs(eType, opts)
	flow, ok := found[f]
	return flow, ok
}

// FindNodeInFlow returns the nodes matching the tag in this flow matching the id and version
func (c *Config) FindNodeInFlow(fRef FlowRef, tag string) []Node {
	for _, f := range c.Flows {
		// first did the flow match
		if f.ID == fRef.ID && f.Ver == fRef.Ver {
			nodes := []Node{}
			// normal tasks
			ns := f.match(NcTask, tag, nil)
			for _, n := range ns {
				nodes = append(nodes, Node(n))
			}
			// merge nodes
			ns = f.match(NcMerge, tag, nil)
			for _, n := range ns {
				nodes = append(nodes, Node(n))
			}
			return nodes
		}
	}
	return nil
}

// zero sets up all the default values
func (c *Config) zero() error {
	for i, f := range c.Flows {
		if err := f.zero(); err != nil {
			return fmt.Errorf("flow %d - %v", i, err)
		}
	}
	return nil
}

// ParseYAML takes a YAML input as a byte array and returns a Config object
// or an error
func ParseYAML(in []byte) (*Config, error) {
	c := &Config{}
	err := yaml.Unmarshal(in, &c)
	if err != nil {
		return c, err
	}
	err = c.zero()
	return c, err
}
