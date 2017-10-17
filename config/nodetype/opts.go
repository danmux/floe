package nodetype

import "encoding/json"

// Opts are the options on the node type that will be compared to those on the event
type Opts map[string]interface{}

func (u *Opts) MarshalJSON() ([]byte, error) {
	f := map[string]string{}
	for k, v := range *u {
		s, ok := v.(string)
		if ok {
			f[k] = s
		} else {
			f[k] = "-"
		}
	}
	return json.Marshal(&f)
}

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
