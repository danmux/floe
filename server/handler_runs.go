package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/floeit/floe/event"
	"github.com/floeit/floe/hub"
)

// this handler returns all runs from all hosts
func hndRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	runs := ctx.hub.AllClientRuns(ctx.ps.ByName("id"))
	return rOK, "", runs
}

// RunSummaries holds slices of RunSummary for each group of run
type RunSummaries struct {
	Active  []RunSummary
	Pending []RunSummary
	Archive []RunSummary
}

// RunSummary represents the state of a run
type RunSummary struct {
	Ref       event.RunRef
	ExecHost  string // the id of the host who's actually executing this run
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Ended     bool
	Good      bool
}

// RunsNewestFirst sorts the runs by most recent start time
type RunsNewestFirst []RunSummary

func (s RunsNewestFirst) Len() int {
	return len(s)
}
func (s RunsNewestFirst) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s RunsNewestFirst) Less(i, j int) bool {
	return s[i].StartTime.Sub(s[j].StartTime) > 0
}

func fromHubRuns(runs hub.Runs) []RunSummary {
	var summaries []RunSummary
	for _, run := range runs {
		summaries = append(summaries, fromHubRun(run))
	}
	sort.Sort(RunsNewestFirst(summaries))
	return summaries
}

func fromHubRun(run *hub.Run) RunSummary {
	return RunSummary{
		Ref:       run.Ref,
		ExecHost:  run.ExecHost,
		StartTime: run.StartTime,
		EndTime:   run.EndTime,
		Ended:     run.Ended,
		Status:    run.Status,
		Good:      run.Good,
	}
}

// this handler answers just for this host and returns the run summaries
func hndP2PRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	id := ctx.ps.ByName("id")
	pending, active, archive := ctx.hub.AllRuns(id)
	summaries := RunSummaries{
		Pending: fromHubRuns(pending),
		Active:  fromHubRuns(active),
		Archive: fromHubRuns(archive),
	}
	return rOK, "", summaries
}
