package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/floeit/floe/client"
	"github.com/floeit/floe/config"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/hub"
)

type field struct {
	ID     string
	Prompt string
	Value  string
}
type runNode struct {
	ID      string
	Name    string
	Type    string
	Enabled bool    // trigger and data only
	Fields  []field // trigger and data only
	Started time.Time
	Stopped time.Time
	Status  string   // "", "running", "finished", "waiting"(for data)
	Result  string   // "success", "failed", "" // only valid when Status="finished"
	Logs    []string // TODO - paging

}

// hndRun answers external call and returns the individual run detail (may come from other host)
func hndRun(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	id := ctx.ps.ByName("id")
	rid := ctx.ps.ByName("rid")

	run := ctx.hub.AllClientFindRun(id, rid)
	if run == nil {
		return rNotFound, "run not found", nil
	}

	// get the config for this run
	conf := ctx.hub.Config()
	flow := conf.Flow(run.Ref.FlowRef)
	if flow == nil {
		return rNotFound, "matching config not found", nil
	}

	graph, problems := flow.Graph()

	triggers := make([]runNode, len(flow.Triggers))

	for i, t := range flow.Triggers {
		rn := runNode{
			ID:   t.ID,
			Name: t.Name,
			Type: t.Type,
		}
		// fill out initiating trigger data - TODO for checkout 'push'
		if t.ID == run.Initiating.SourceNode.ID && t.Type == "data" {
			form, ok := t.Opts["form"].(map[string]interface{})
			values := run.Initiating.Opts
			if ok {
				for _, fld := range form["fields"].([]interface{}) {
					f := fld.(map[string]interface{})
					id := f["id"].(string)
					rn.Fields = append(rn.Fields, field{
						ID:     id,
						Prompt: f["prompt"].(string),
						Value:  values[id].(string),
					})
				}
			}
			rn.Enabled = true
		}
		triggers[i] = rn
	}

	response := struct {
		FlowName string
		Name     string
		Triggers []runNode
		Graph    [][]runNode
		Problems []string
	}{
		FlowName: flow.Name,
		Name:     flow.Name + " " + run.Ref.Run.String(),
		Triggers: triggers,
		Graph:    buildRunResp(graph[1:], flow, run),
		Problems: problems,
	}

	return rOK, "", response
}

func buildRunResp(graph [][]string, conf *config.Flow, run *client.Run) [][]runNode {
	nodes := make([][]runNode, len(graph))
	for i, gns := range graph {
		nodes[i] = make([]runNode, len(graph[i]))
		for j, id := range gns {
			cn := conf.Node(id)
			if cn == nil {
				continue
			}
			rn := runNode{
				ID:   id,
				Name: cn.Name,
				Type: cn.Type,
			}
			if cn.Class != "task" {
				nodes[i][j] = rn
				continue
			}

			switch cn.Type {
			case "data":
				res := run.DataNodes[id]
				rn.Enabled = res.Enabled
				rn.Started = res.Started
				rn.Stopped = res.Stopped
				switch {
				case !rn.Stopped.IsZero():
					rn.Status = "finished"
				case !rn.Started.IsZero():
					rn.Status = "waiting"
				}
				// TODO - construct the fields - for entry - or for reporting
				// using res.Opts and cn
			default:
				res := run.ExecNodes[id]
				rn.Logs = res.Logs
				rn.Started = res.Started
				rn.Stopped = res.Stopped
				switch {
				case !rn.Stopped.IsZero():
					rn.Status = "finished"
					if res.Good {
						rn.Result = "success"
					} else {
						rn.Result = "failed"
					}
				case !rn.Started.IsZero():
					rn.Status = "running"
				}
			}
			nodes[i][j] = rn
		}
	}
	return nodes
}

// hndP2PRun answers internal calls just for this host and returns the individual run detail
func hndP2PRun(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	id := ctx.ps.ByName("id")
	rid := ctx.ps.ByName("rid")
	run := ctx.hub.FindRun(id, rid)
	if run == nil {
		return rNotFound, "not found", nil
	}
	return rOK, "", run
}

// hndP2PRuns answers internal calls just for this host and returns the run summaries
func hndP2PRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	flowID := ctx.ps.ByName("id")
	pending, active, archive := ctx.hub.AllRuns(flowID)
	summaries := RunSummaries{
		Pending: fromHubRuns(pending),
		Active:  fromHubRuns(active),
		Archive: fromHubRuns(archive),
	}
	return rOK, "", summaries
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
		// TODO - add if waiting for data
	}
}
