package par

// Status defines the possible task statuses
type Status int

// the values that Status can have
const (
	StSuccess Status = iota // 0
	StFail                  // 1

	KeyTidyDesk = "reset_workspace" // reset or keep the workspace between executions
)

// Props defines our property object
type Props map[string]string

// Params is created per executing thread of a floe - to set up some defaults. Then a Params object is cloned for each executing task.
// Props are copied in and mutated as the floe progresses, so a downstream task can access params set by an upstream.
type Params struct {
	TaskID     string // the unique (within the floe)
	Complete   bool   // set true on complete tasks
	Status     Status // task status as defined above
	ExitStatus int    // capture the exit status of any os task - or set explicitly by custom tasks to trigger downstream logic
	Response   string // snappy single liner summarising the result of the task
	Props      Props  // the set of key values that can be mutated by each task and accessed anywhere else in the floe

	floeID   string // the id of the floe this belongs to
	threadID int    // the thread number that this object belongs to
}

// Copy copies the salient fields into the receiver
func (p *Params) Copy(ip *Params) {
	// reproduce the id
	p.floeID = ip.floeID
	p.threadID = ip.threadID
	p.TaskID = ip.TaskID
	p.Complete = false // just to make sure

	// and the other info stuff
	p.Props = ip.Props
}
