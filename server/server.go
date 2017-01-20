package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/hub"
	"github.com/floeit/floe/log"
)

const rootPath = "/build/api"

// LaunchWeb sets up all the http routes runs the server and launches the trigger flows
// rp is the root path
func LaunchWeb(host, rp string, hub *hub.Hub) {
	if rp == "" {
		rp = rootPath
	}
	r := httprouter.New()
	r.HandleMethodNotAllowed = false
	r.NotFound = notFoundHandler{}
	r.PanicHandler = panicHandler

	h := handler{hub: hub}

	// --- general ---
	r.GET(rp+"/config", h.mw(confHandler, true))

	// --- authentication ---
	r.POST(rp+"/login", h.mw(loginHandler, false))
	r.POST(rp+"/logout", h.mw(logoutHandler, true))

	// --- api ---
	r.GET(rp+"/flows", h.mw(hndAllFlows, true))

	// --- subscription endpoints ---
	h.setupSubs(rp+"/subs/", r, hub)

	// --- CORS ---
	r.OPTIONS(rp+"/*all", h.mw(nil, false)) // catch all options

	/*
		r.GET(rp+"/flows/:flid", h.mw(floeHandler, true))
		r.POST(rp+"/flows/:flid/exec", h.mw(execHandler, true))
		r.POST(rp+"/flows/:flid/stop", h.mw(stopHandler, true))
		r.GET(rp+"/flows/:flid/run/:agentid/:runid", h.mw(runHandler, true)) // get the current progress of a run for an agent and run

		// --- web socket connection ---
		r.GET(rp+"/msg", wsHandler)



		// --- the web page stuff ---
		r.GET("/build/", indexHandler)
		r.ServeFiles("/build/css/*filepath", http.Dir("public/build/css"))
		r.ServeFiles("/build/fonts/*filepath", http.Dir("public/build/fonts"))
		r.ServeFiles("/build/img/*filepath", http.Dir("public/build/img"))
		r.ServeFiles("/build/js/*filepath", http.Dir("public/build/js"))

	*/

	log.Info("agent server starting on:", host)
	log.Fatal(http.ListenAndServe(host, r))
}
