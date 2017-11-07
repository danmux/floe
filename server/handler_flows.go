package server

import (
	"net/http"

	"github.com/floeit/floe/config"

	"github.com/floeit/floe/hub"
)

func hndAllFlows(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.Config()
}

func hndFlow(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	id := ctx.ps.ByName("id")

	// config
	conf := ctx.hub.Config()
	latest := conf.LatestFlow(id)

	// runs
	pending, active, archive := ctx.hub.AllRuns(id)
	summaries := RunSummaries{
		Pending: fromHubRuns(pending),
		Active:  fromHubRuns(active),
		Archive: fromHubRuns(archive),
	}

	response := struct {
		Config *config.Flow
		Runs   RunSummaries
	}{
		Config: latest,
		Runs:   summaries,
	}

	return rOK, "", response
}

func hndP2PExecFlow(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {

	t := hub.Todo{}
	if ok, code, msg := decodeBody(rw, r, &t); !ok {
		return code, msg, nil
	}

	ok, err := ctx.hub.ExecutePending(t)
	if err != nil {
		return rErr, err.Error(), nil
	}
	if !ok {
		return rConflict, "host has resource conflicting active flows", nil
	}

	return rOK, "started", nil
}
