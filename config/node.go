package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
// TODO cleanup - only used once
type nid interface {
	setID(string)
	setName(string)
	name() string
	id() string
}

// Node is the interface through which the task is accessed
// TODO so far looks like we only have a common task structure so this may not be needed
type Node interface {
	FlowRef() FlowRef
	NodeRef() NodeRef
	Class() NodeClass
	Execute(nt.Workspace, nt.Opts) (int, nt.Opts, error)
	Status(status int) (string, bool)
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
	Good       []int   // the array of exit status codes considered a success
	IgnoreFail bool    `yaml:"ignore-fail"` // only ever send the good event cant be used in conjunction with UseStatus
	UseStatus  bool    `yaml:"use-status"`  // use status if we don't send good or bad but the actual status code as an event
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

// Status will return the string to use on an event tag and a boolean to
// indicate if the status is considered good
func (t *task) Status(status int) (string, bool) {
	// always good if ignore fail
	if t.IgnoreFail {
		return "good", true
	}
	// is this code considered a success
	good := false
	// no specific good statuses so consider 0 success, all others fail
	if len(t.Good) == 0 {
		good = status == 0
	} else {
		for _, s := range t.Good {
			if s == status {
				good = true
				break
			}
		}
	}
	// use specific exit statuses
	if t.UseStatus {
		return strconv.Itoa(status), good
	}
	// or binary result
	if good {
		return "good", true
	}
	return "bad", false
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

func (t *task) matchedSub(eType string, opts *nt.Opts) bool {
	// subs matches must always have opts
	if opts == nil {
		return false
	}
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

func (t *task) matched(tag string) bool {
	// match on the Listen
	if t.Listen != "" && t.Listen == tag {
		return true
	}
	// or if any tags in the the Wait list match (merge nodes only)
	for _, wt := range t.Wait {
		if wt == tag {
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

	n := nt.GetNodeType(t.Type)
	if n == nil {
		return nil
	}

	n.CastOpts(&t.Opts)

	return nil
}
