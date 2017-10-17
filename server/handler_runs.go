package server

import (
	"net/http"
)

func hndActiveRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.RunsActive()
}

func hndPendingRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.RunsPending()
}

func hndArchiveRuns(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.RunsActive()
}
