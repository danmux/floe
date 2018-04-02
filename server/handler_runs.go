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

	response := struct {
		Config   *config.Flow
		Graph    [][]string
		Problems []string
		Run      *client.Run
	}{
		Config:   flow,
		Graph:    graph[1:],
		Problems: problems,
		Run:      run,
	}

	return rOK, "", response
}

/*
{
	"Message": "OK",
	"Payload": {
	  "Config": {
		"ID": "build-project",
		"Ver": 1,
		"Name": "Build Project",
		"ReuseSpace": true,
		"HostTags": [
		  "linux",
		  "go",
		  "couch"
		],
		"ResourceTags": [
		  "couchbase",
		  "nic"
		],
		"Triggers": [
		  {
			"Class": "trigger",
			"Ref": {
			  "Class": "trigger",
			  "ID": "push"
			},
			"ID": "push",
			"Name": "push",
			"Listen": "",
			"Wait": null,
			"Type": "git-push",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "url": "blah.blah"
			}
		  },
		  {
			"Class": "trigger",
			"Ref": {
			  "Class": "trigger",
			  "ID": "start"
			},
			"ID": "start",
			"Name": "start",
			"Listen": "",
			"Wait": null,
			"Type": "data",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "form": {
				"fields": [
				  {
					"id": "from_hash",
					"prompt": "From Branch (or hash)",
					"type": "text"
				  },
				  {
					"id": "to_hash",
					"prompt": "To Branch (or hash)",
					"type": "text"
				  }
				],
				"title": "Start"
			  }
			}
		  }
		],
		"Tasks": [
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "checkout"
			},
			"ID": "checkout",
			"Name": "checkout",
			"Listen": "trigger.good",
			"Wait": null,
			"Type": "git-merge",
			"Good": [
			  0
			],
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": null
		  },
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "echo"
			},
			"ID": "echo",
			"Name": "echo",
			"Listen": "task.checkout.good",
			"Wait": null,
			"Type": "exec",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "cmd": "echo dan",
			  "file": "BUILD.floe"
			}
		  },
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "build"
			},
			"ID": "build",
			"Name": "build",
			"Listen": "task.checkout.good",
			"Wait": null,
			"Type": "exec",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "cmd": "make build"
			}
		  },
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "test"
			},
			"ID": "test",
			"Name": "Test",
			"Listen": "task.build.good",
			"Wait": null,
			"Type": "exec",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "cmd": "make test"
			}
		  },
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "sign-off"
			},
			"ID": "sign-off",
			"Name": "Sign Off",
			"Listen": "task.build.good",
			"Wait": null,
			"Type": "data",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": {
			  "form": {
				"fields": [
				  {
					"id": "tests_passed",
					"prompt": "Did the manual testing pass?",
					"type": "bool"
				  },
				  {
					"id": "to_hash",
					"prompt": "To Branch (or hash)",
					"type": "string"
				  }
				],
				"title": "Sign off Manual Testing"
			  }
			}
		  },
		  {
			"Class": "merge",
			"Ref": {
			  "Class": "merge",
			  "ID": "signed"
			},
			"ID": "signed",
			"Name": "wait test and sign off",
			"Listen": "",
			"Wait": [
			  "task.echo.good",
			  "task.test.good",
			  "task.sign-off.good"
			],
			"Type": "all",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": null
		  },
		  {
			"Class": "task",
			"Ref": {
			  "Class": "task",
			  "ID": "complete"
			},
			"ID": "complete",
			"Name": "complete",
			"Listen": "merge.signed.good",
			"Wait": null,
			"Type": "end",
			"Good": null,
			"IgnoreFail": false,
			"UseStatus": false,
			"Opts": null
		  }
		]
	  },
	  "Graph": [
		[
		  "checkout"
		],
		[
		  "echo",
		  "build"
		],
		[
		  "test",
		  "sign-off"
		],
		[
		  "signed"
		],
		[
		  "complete"
		]
	  ],
	  "Problems": null,
	  "Run": {
		"Ref": {
		  "FlowRef": {
			"ID": "build-project",
			"Ver": 1
		  },
		  "Run": {
			"HostID": "h1",
			"ID": 1
		  },
		  "ExecHost": "h1"
		},
		"ExecHost": "",
		"StartTime": "2018-04-01T07:58:36.003761394+01:00",
		"EndTime": "0001-01-01T00:00:00Z",
		"Ended": false,
		"Status": "",
		"Good": false,
		"MergeNodes": {
		  "signed": {
			"Waits": {
			  "task.echo.good": true,
			  "task.test.good": true
			},
			"Opts": {}
		  }
		},
		"DataNodes": {
		  "sign-off": {
			"Enabled": true,
			"Opts": {
			  "form": {
				"fields": [
				  {
					"id": "tests_passed",
					"prompt": "Did the manual testing pass?",
					"type": "bool"
				  },
				  {
					"id": "to_hash",
					"prompt": "To Branch (or hash)",
					"type": "string"
				  }
				],
				"title": "Sign off Manual Testing"
			  }
			}
		  }
		},
		"ExecNodes": {
		  "build": {
			"Opts": null,
			"Logs": [
			  "something after 0 seconds",
			  "something after 1 seconds",
			  "something after 2 seconds",
			  "something after 3 seconds",
			  "something after 4 seconds"
			]
		  },
		  "echo": {
			"Opts": null,
			"Logs": [
			  "something after 0 seconds",
			  "something after 1 seconds",
			  "something after 2 seconds",
			  "something after 3 seconds",
			  "something after 4 seconds"
			]
		  },
		  "test": {
			"Opts": null,
			"Logs": [
			  "something after 0 seconds",
			  "something after 1 seconds",
			  "something after 2 seconds",
			  "something after 3 seconds",
			  "something after 4 seconds"
			]
		  }
		}
	  }
	}
  }
*/

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
