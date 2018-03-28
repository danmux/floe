package nodetype

// data
type data struct{}

func (d data) Match(qs, as Opts) bool {
	return true
}

func (d data) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {
	// data nodes just fill in the opts
	return 0, in, nil
}
