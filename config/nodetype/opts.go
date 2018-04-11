package nodetype

// Workspace is anything specific to a workspace for a single run
type Workspace struct {
	BasePath string
}

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

func (o Opts) int(key string) (int, bool) {
	si, ok := o[key]
	if !ok {
		return 0, false
	}
	s, ok := si.(int)
	if !ok {
		fs, fok := si.(float64)
		if !fok {
			return 0, false
		}
		return int(fs), true
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

// MergeOpts merges l and r into a new Opts struct, r taking precedence
func MergeOpts(l, r Opts) Opts {
	o := Opts{}
	for k, v := range l {
		o[k] = v
	}
	for k, v := range r {
		// most arrays will be a full replacement, but environment variables get appended
		if k == "env" {
			if v1, ok := o[k]; ok {
				o[k] = appendIfArr(v1, v)
			}
		} else {
			o[k] = v
		}
	}
	return o
}

// appendIfArr appends r to l, if they are both []string else any one of them that is []string
// is returned, else nil is returned
func appendIfArr(l interface{}, r interface{}) []interface{} {
	la, lok := l.([]interface{})
	ra, rok := r.([]interface{})

	if !lok && !rok {
		return nil
	}
	if !lok && rok {
		return ra
	}
	if lok && !rok {
		return la
	}
	return append(la, ra...)
}

// Fixup allows the receiver to be able to be rendered as json
func (o *Opts) Fixup() {
	for k, v := range *o {
		(*o)[k] = yamlToJSON(v)
	}
}

// yamlToJSON takes the generic Yaml maps with interface keys
// and converts them into the json string based keys
func yamlToJSON(in interface{}) interface{} {
	// TODO - consider mapstructure
	if m, ok := in.(map[interface{}]interface{}); ok {
		o := map[string]interface{}{}
		for k, v := range m {
			o[k.(string)] = yamlToJSON(v)
		}
		return o
	}
	if m, ok := in.([]interface{}); ok {
		o := make([]interface{}, len(m))
		for i, v := range m {
			o[i] = yamlToJSON(v)
		}
		return o
	}

	return in
}
