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

func (u *Opts) Fixup() {
	for k, v := range *u {
		(*u)[k] = yamlToJson(v)
	}
}

// yamlToJson takes the generic Yaml maps with interface keys
// and converts them into the json string based keys
func yamlToJson(in interface{}) interface{} {
	// TODO - consider mapstructure
	if m, ok := in.(map[interface{}]interface{}); ok {
		o := map[string]interface{}{}
		for k, v := range m {
			o[k.(string)] = yamlToJson(v)
		}
		return o
	}
	if m, ok := in.([]interface{}); ok {
		o := make([]interface{}, len(m))
		for i, v := range m {
			o[i] = yamlToJson(v)
		}
		return o
	}

	return in
}
