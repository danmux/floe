package server

import (
	"net/http"

	"github.com/floeit/floe/hub"
)

func hndAllFlows(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.Config()
}

func hndExecFlow(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {

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
