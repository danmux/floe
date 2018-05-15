package nodetype

type timer struct{}

func (d timer) Match(qs, as Opts) bool {
	qp, ok := qs.int("period")
	if !ok {
		return false
	}

	ap, ok := as.int("period")
	if !ok {
		return false
	}

	return qp == ap
}

// Execute
func (d timer) Execute(ws *Workspace, in Opts, output chan string) (int, Opts, error) {

	return 0, in, nil
}
