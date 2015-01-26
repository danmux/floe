package flow

type Props map[string]string

type Params struct {
	FlowName   string // these three make up a unique ID for the task
	ThreadId   int
	TaskId     string
	TaskName   string
	Complete   bool // set true on complete tasks
	TaskType   string
	Status     int
	ExitStatus int
	Response   string
	Props      Props
	Raw        []byte
}

func MakeParams() *Params {
	return &Params{
		Props: Props{KEY_WORKSPACE: "workspace"}, // default workspace name is .... well ... workspace
	}
}

func (p *Params) Copy(ip *Params) {
	// reproduce the id
	p.FlowName = ip.FlowName
	p.ThreadId = ip.ThreadId
	p.TaskName = ip.TaskName
	p.TaskId = ip.TaskId
	p.Complete = false // just to make sure

	// and the other info stuff
	p.TaskType = ip.TaskType
	p.Props = ip.Props
}
