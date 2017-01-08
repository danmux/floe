package server

import "net/http"

func hndAllFlows(w http.ResponseWriter, req *http.Request, ctx *context) (int, string, renderable) {
	return rOK, "", ctx.hub.Config()
}
