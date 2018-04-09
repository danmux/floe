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
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
	Value  string `json:"value"`
}

type runNode struct {
	ID      string
	Name    string
	Class   config.NodeClass
	Type    string
	Enabled bool    // trigger and data only
	Fields  []field // trigger and data only
	Started time.Time
	Stopped time.Time
	Status  string          // "", "running", "finished", "waiting"(for data)
	Result  string          // "success", "failed", "" // only valid when Status="finished"
	Logs    []string        // TODO - paging
	Waits   map[string]bool // the events the merge node has seen
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
			buildFields(&rn, t.Opts, run.Initiating.Opts)
			rn.Enabled = true
		}
		triggers[i] = rn
	}

	response := struct {
		FlowName string
		Name     string
		Triggers []runNode
		Graph    [][]runNode
		Summary  RunSummary
		Problems []string
	}{
		FlowName: flow.Name,
		Name:     flow.Name + " " + run.Ref.Run.String(),
		Triggers: triggers,
		Graph:    buildRunResp(graph[1:], flow, run),
		Summary: RunSummary{
			Ref:       run.Ref,
			ExecHost:  run.ExecHost,
			Status:    runStatus(run.StartTime, run.Ended, run.Good),
			StartTime: run.StartTime,
			EndTime:   run.EndTime,
			Ended:     run.Ended,
			Good:      run.Good,
		},
		Problems: problems,
	}

	return rOK, "", response
}

/*
(nodetype.Opts) (len=2) {
 (string) (len=4) "form": (map[string]interface {}) (len=2) {
  (string) (len=6) "fields": ([]interface {}) (len=2 cap=2) {
   (map[string]interface {}) (len=4) {
    (string) (len=2) "id": (string) (len=12) "tests_passed",
    (string) (len=6) "prompt": (string) (len=28) "Did the manual testing pass?",
    (string) (len=4) "type": (string) (len=4) "bool",
    (string) (len=5) "value": (string) (len=3) "fds"
   },
   (map[string]interface {}) (len=4) {
    (string) (len=2) "id": (string) (len=7) "to_hash",
    (string) (len=6) "prompt": (string) (len=19) "To Branch (or hash)",
    (string) (len=4) "type": (string) (len=6) "string",
    (string) (len=5) "value": (string) (len=14) "ttrtrtrtrtrtrt"
   }
  },
  (string) (len=5) "title": (string) (len=23) "Sign off Manual Testing"
 },
 (string) (len=6) "values": (map[string]interface {}) (len=2) {
  (string) (len=7) "to_hash": (string) (len=14) "ttrtrtrtrtrtrt",
  (string) (len=12) "tests_passed": (string) (len=3) "fds"
 }
}
*/

// buildFields uses the node config Opts and any current values
// and creates a set of Fields from it
func buildFields(rn *runNode, confOpts, values map[string]interface{}) {
	// TODO - consider mapstructure
	form, ok := confOpts["form"].(map[string]interface{})
	if !ok {
		return
	}
	for _, fld := range form["fields"].([]interface{}) {
		f := fld.(map[string]interface{})
		id := f["id"].(string)
		val := ""
		vi, ok := values[id]
		if ok {
			val = vi.(string)
		}
		rn.Fields = append(rn.Fields, field{
			ID:     id,
			Prompt: f["prompt"].(string),
			Value:  val,
		})
	}
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
				ID:    id,
				Name:  cn.Name,
				Class: cn.Class,
				Type:  cn.Type,
			}

			if cn.Class == "merge" {
				res := run.MergeNodes[id]
				rn.Waits = res.Waits
				if rn.Waits == nil {
					rn.Waits = map[string]bool{}
				}
				for _, w := range cn.Wait {
					if _, ok := rn.Waits[w]; !ok {
						rn.Waits[w] = false
					}
				}
				rn.Started = res.Started
				rn.Stopped = res.Stopped
				switch {
				case !rn.Stopped.IsZero():
					rn.Status = "finished"
				case !rn.Started.IsZero():
					rn.Status = "waiting"
				}
				nodes[i][j] = rn
				continue
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
				vals := map[string]interface{}{}
				if v, ok := res.Opts["values"]; ok {
					vals = v.(map[string]interface{})
				}
				buildFields(&rn, cn.Opts, vals)
			default:
				res := run.ExecNodes[id]
				rn.Logs = res.Logs
				rn.Started = res.Started
				rn.Stopped = res.Stopped
				switch {
				case !rn.Stopped.IsZero():
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
	Ref      event.RunRef
	ExecHost string // the id of the host who's actually executing this run
	Status   string // TODO include if waiting for data
	// TODO add branch/tag/hash
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

func runStatus(startTime time.Time, ended, good bool) string {
	status := "pendind"
	if !startTime.IsZero() { // if it has a start time
		status = "running"
		if ended {
			if good {
				status = "good"
			} else {
				status = "bad"
			}
		}
	}
	return status
}

func fromHubRun(run *hub.Run) RunSummary {
	return RunSummary{
		Ref:       run.Ref,
		ExecHost:  run.ExecHost,
		StartTime: run.StartTime,
		EndTime:   run.EndTime,
		Ended:     run.Ended,
		Status:    runStatus(run.StartTime, run.Ended, run.Good),
		Good:      run.Good,
		// TODO - add branch
		// TODO - add if waiting for data
	}
}
