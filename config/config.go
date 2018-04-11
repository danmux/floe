package config

import (
	"fmt"

	"gopkg.in/yaml.v2"

	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/log"
)

// Config is the set of nodes and rules
type Config struct {
	Common commonConfig
	// the list of flow configurations
	Flows []*Flow
}

type commonConfig struct {
	// all other floe Hosts
	Hosts []string
	// the api base url - in case hosting on a sub domain
	BaseURL string `yaml:"base-url"`
	// ConfigPath is a path to the config which can be a path to a file in a git repo
	// e.g. git@github.com:floeit/floe.git/build/FLOE.yaml
	ConfigPath string `yaml:"config-path"`

	// StoreType define which type of store to use
	StoreType string `yaml:"store-type"` // memory, local, ec2

	// TODO ec2 - or back to github
	// Store Root is ec2 bucket path
	// StoreRoot string `yaml:"store-root"`

	// StoreCredentials is a string in some format or other to provide needed credentials for
	// specific store type.
	// StoreCredentials string `yaml:"store-credentials"`
}

// FoundFlow is a struct containing a cut down set of properties of a Flow.
// It can be used to decide on the best host to use to run this Flow.
type FoundFlow struct {
	Ref          FlowRef
	ReuseSpace   bool
	ResourceTags []string
	HostTags     []string
	Env          []string

	Nodes []*node // nodes that matched the criteria to have fund the node
}

// FindFlowsByTriggers finds all flows where its subs match the given params
func (c *Config) FindFlowsByTriggers(eType string, flow FlowRef, opts nt.Opts) map[FlowRef]FoundFlow {
	res := map[FlowRef]FoundFlow{}
	for _, f := range c.Flows {
		// if a flow is specified it has to match
		if flow.NonZero() {
			log.Debugf("config - comparing flow:<%s> to config flow:<%s-%d>", flow, f.ID, f.Ver)
			if f.ID != flow.ID || f.Ver != flow.Ver {
				continue
			}
		}
		log.Debugf("config - found flow: <%s-%d>. %d triggers", f.ID, f.Ver, len(f.Triggers))
		// match on other stuff
		ns := f.matchTriggers(eType, &opts)
		// found some matching nodes for this flow
		if len(ns) > 0 {
			if len(ns) > 1 {
				log.Warning("triggered flow has too many triggers, using first", f.ID, f.Ver, len(f.Triggers))
			}
			// make sure this flow is in the results
			fr := ns[0].FlowRef()
			ff, ok := res[fr]
			if !ok {
				ff = FoundFlow{
					Ref:          fr,
					ReuseSpace:   f.ReuseSpace,
					ResourceTags: f.ResourceTags,
					HostTags:     f.HostTags,
					Env:          f.Env,
				}
			}
			ff.Nodes = []*node{ns[0]} // use the first one
			res[fr] = ff
		} else {
			log.Debugf("config - flow:<%s> failed on trigger match", flow)
		}
	}
	return res
}

// FindFlow finds the specific flow where its subs match the given params
func (c *Config) FindFlow(f FlowRef, eType string, opts nt.Opts) (FoundFlow, bool) {
	found := c.FindFlowsByTriggers(eType, f, opts)
	flow, ok := found[f]
	return flow, ok
}

// Flow returns the flow config matching the id and version
func (c *Config) Flow(fRef FlowRef) *Flow {
	for _, f := range c.Flows {
		if f.ID == fRef.ID && f.Ver == fRef.Ver {
			return f
		}
	}
	return nil
}

// LatestFlow returns the flow config matching the id with the highest version
func (c *Config) LatestFlow(id string) *Flow {
	var latest *Flow
	highestVer := 0
	for _, f := range c.Flows {
		if f.ID != id {
			continue
		}
		if f.Ver > highestVer {
			latest = f
		}
	}
	return latest
}

// FindNodeInFlow returns the nodes matching the tag in this flow matching fRef
// returns false if the flow is not found
func (c *Config) FindNodeInFlow(fRef FlowRef, tag string) (FoundFlow, bool) {
	ff := FoundFlow{}
	f := c.Flow(fRef)
	if f == nil {
		return ff, false
	}
	// we found the matching flow so can find any matching nodes
	return FoundFlow{
		Ref:          fRef,
		ReuseSpace:   f.ReuseSpace,
		ResourceTags: f.ResourceTags,
		HostTags:     f.HostTags,
		Nodes:        f.matchTag(tag),
	}, true
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
