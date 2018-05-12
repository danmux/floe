package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cavaliercoder/grab"

	nt "github.com/floeit/floe/config/nodetype"
	yaml "gopkg.in/yaml.v2"
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

	// FlowFile is a path to a config file describing the Tasks.
	// It can be a path to a file in a git repo e.g. git@github.com:floeit/floe.git/build/FLOE.yaml
	// or a local file e.g. file:./foo-bar/floe.yaml
	// a FlowFile may override any setting from the flows defined in the main config file, but it
	// does not make much sense that they override the Triggers.
	// If this file is is taken from the same repo as the first `git-checkout`
	FlowFile string `yaml:"flow-file"`

	Name         string   // human friendly name
	ReuseSpace   bool     `yaml:"reuse-space"`   // if true then will use the single workspace and will mutex with other instances of this Flow
	HostTags     []string `yaml:"host-tags"`     // tags that must match the tags on the host
	ResourceTags []string `yaml:"resource-tags"` // tags that if any flow is running with any matching tags then don't launch
	Env          []string // key=value environment variables with

	// Triggers are the node types that define how a run is triggered for this flow.
	Triggers []*node

	// The things to do once a trigger has started this flow
	Tasks []*node
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

// MatchTag finds all nodes that are waiting for this event tag
func (f *Flow) MatchTag(tag string) []*node {
	res := []*node{}
	for _, s := range f.Tasks {
		if s.matched(tag) {
			res = append(res, s)
		}
	}
	return res
}

// Load looks at the FlowFile and loads in the flow from that reference
// overriding any pre-existing settings, except triggers.
// The ref param can be used to replace any '{{ref}}' in the FlowFile
func (f *Flow) Load(cacheDir, ref string) (err error) {
	if f.FlowFile == "" {
		return nil
	}
	f.FlowFile = strings.Replace(f.FlowFile, "{{ref}}", ref, -1)
	var content []byte
	switch getRefType(f.FlowFile) {
	case "local":
		content, err = ioutil.ReadFile(f.FlowFile)
	case "web":
		content, err = get(cacheDir, f.FlowFile)
	default:
		return fmt.Errorf("unrecognised floe file type: <%s>", f.FlowFile)
	}
	if err != nil {
		return err
	}

	// unmarshal into a flow
	newFlow := &Flow{}
	err = yaml.Unmarshal(content, &newFlow)
	if err != nil {
		return err
	}

	// set up the flow, and copy bits into this flow
	err = newFlow.zero()
	if err != nil {
		return err
	}
	if len(newFlow.Name) != 0 {
		f.Name = newFlow.Name
	}
	f.ReuseSpace = newFlow.ReuseSpace
	if len(newFlow.HostTags) != 0 {
		f.HostTags = newFlow.HostTags
	}
	if len(newFlow.ResourceTags) != 0 {
		f.ResourceTags = newFlow.ResourceTags
	}
	if len(newFlow.Env) != 0 {
		f.Env = newFlow.Env
	}
	if len(newFlow.Tasks) != 0 {
		f.Tasks = newFlow.Tasks
	}
	// Pointless overriding triggers - as they are what caused this load
	return nil
}

// get gets the file from the web or the cache and returns its contents
func get(cacheDir, url string) ([]byte, error) {
	client := grab.NewClient()
	req, err := grab.NewRequest(cacheDir, url)
	if err != nil {
		return nil, err
	}
	resp := client.Do(req)
	<-resp.Done
	return ioutil.ReadFile(resp.Filename)
}

// getRefType returns the reference type:
// "web" - it is fetchable from the web,
// "git" - it can be got from a repo,
// "local" - it can be got from the local files system
func getRefType(fileRef string) string {
	if strings.Contains(fileRef, "git@") {
		return "git"
	}
	if strings.HasPrefix(fileRef, "http") {
		return "web"
	}
	return "local"
}

// zero checks and corrects ids and changes the options format from yaml default to
// one compatible with json serialising
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

	// check for unique id's
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

func (f *Flow) matchTriggers(eType string, opts *nt.Opts) []*node {
	res := []*node{}
	for _, s := range f.Triggers {
		if s.matchedTriggers(eType, opts) {
			res = append(res, s)
		}
	}
	return res
}

// methods that implement nid so the flow can be zeroNid'd
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
