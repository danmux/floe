package server

import (
	"net/http"
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
	StartTime time.Time
	EndTime   time.Time
	Ended     bool
	Status    string
	Good      bool
}

func fromHubRuns(runs hub.Runs) []RunSummary {
	var summaries []RunSummary
	for _, run := range runs {
		summaries = append(summaries, fromHubRun(run))
	}
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
	summaries := RunSummaries{
		Active:  fromHubRuns(ctx.hub.RunsActive()),
		Pending: fromHubRuns(ctx.hub.RunsPending()),
		Archive: fromHubRuns(ctx.hub.RunsArchive()),
	}
	return rOK, "", summaries
}
