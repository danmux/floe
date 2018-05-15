package nodetype

import (
	"github.com/mitchellh/mapstructure"
)

type data struct{}

func (d data) Match(qs, as Opts) bool {
	return true
}

type dataOpts struct {
	Values map[string]string `json:"values"`
	Form   form              `json:"form"`
}

type form struct {
	Title  string  `json:"title"`
	Fields []field `json:"fields"`
}

type field struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
	Type   string `json:"type"`
	Value  string `json:"value"`
}

// Execute on data nodes fill in the opts, validate the form, and decide if the node can be considered
// good or bad.
// returns status 0 = form requirements met, 1 = an error (error will be set), 2 = needs more data
func (d data) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {
	do := dataOpts{}

	err := mapstructure.Decode(in, &do)
	if err != nil {
		return 1, nil, err
	}

	rCode := 0
	for i, f := range do.Form.Fields {
		if v, ok := do.Values[f.ID]; ok {
			f.Value = v
			do.Form.Fields[i] = f
			// TODO add and validate mandatory fields
			// TODO look for single field representing good or bad
		} else {
			rCode = 2
		}
	}

	out := map[string]interface{}{
		"form":   do.Form,
		"values": do.Values,
	}

	return rCode, out, nil
}
