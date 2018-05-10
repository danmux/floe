package server

import (
	"net/http"

	"github.com/floeit/floe/client"
	"github.com/floeit/floe/config"
	"github.com/floeit/floe/hub"
)

func hndAllFlows(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.Config()
}

// hndFlow returns the latest config and run summaries from all clients for this flow
func hndFlow(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	id := ctx.ps.ByName("id")

	// get the latest config
	conf := ctx.hub.Config()
	latest := conf.LatestFlow(id)
	if latest == nil {
		return rNotFound, "not found", nil
	}

	// and run summaries from all hosts
	summaries := ctx.hub.AllClientRuns(id)

	response := struct {
		Config *config.Flow
		Runs   client.RunSummaries
	}{
		Config: latest,
		Runs:   summaries,
	}

	return rOK, "", response
}

// hndP2PExecFlow is the handler for the internal call to execute the flow on this node
func hndP2PExecFlow(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {

	pend := hub.Pend{}
	if ok, code, msg := decodeBody(rw, r, &pend); !ok {
		return code, msg, nil
	}

	ok, err := ctx.hub.ExecutePending(pend)
	if err != nil {
		return rErr, err.Error(), nil
	}
	if !ok {
		return rConflict, "host has resource conflicting active flows", nil
	}

	return rOK, "started", nil
}
