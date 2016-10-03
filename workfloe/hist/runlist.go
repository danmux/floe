package hist

import (
	"strconv"
	"time"

	"github.com/floeit/floe/workfloe/store"
)

const (
	recType = "run"
	idxType = "index"
)

type summary struct {
	RunID     int
	Reason    string
	Error     string
	Start     time.Time
	Duration  int
	Completed bool
}

// RunList keeps track of all runs for a particular launcher
type RunList struct {
	MaxUsedID int
	FloeID    string    // matches the floe Id
	Summaries []summary // to serialise some salient points about each run

	runs []*LaunchResult // a set of response stats by task id in our workfloe for the last run
}

func (rl *RunList) AddRun(result *LaunchResult) {
	// don't add one that is already in the list (if RunID is set then it must have been added)
	if result.RunID > 0 {
		return
	}
	len := len(rl.runs)
	if len > rl.MaxUsedID {
		rl.MaxUsedID = len
	}

	rl.MaxUsedID++

	result.RunID = rl.MaxUsedID
	rl.runs = append(rl.runs, result)
}

func (rl *RunList) Save(store store.Store) error {
	// the latest is always the last in the list
	latest := rl.runs[len(rl.runs)-1]
	// update the summaries
	// make sure we have same number of summaries as runs
	if len(rl.Summaries) < len(rl.runs) {
		ts := make([]summary, len(rl.runs)-len(rl.Summaries))
		rl.Summaries = append(rl.Summaries, ts...)
	}

	rl.Summaries[len(rl.Summaries)-1] = summary{
		RunID:     latest.RunID,
		Error:     latest.Error,
		Start:     latest.Start,
		Duration:  latest.Duration,
		Completed: latest.Completed,
	}
	// save the summaries
	if err := store.Set(recType, idxType, rl); err != nil {
		return err
	}
	// save the latest
	return store.Set(strconv.Itoa(latest.RunID), recType, latest)
}

// Load sets up the runList public fields from store
func (rl *RunList) Load(store store.Store) error {
	if err := store.Get(recType, idxType, rl); err != nil {
		return err
	}
	// make space for the runs which will be lazy loaded as required
	rl.runs = make([]*LaunchResult, len(rl.Summaries))
	return nil
}
