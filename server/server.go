package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/hub"
	"github.com/floeit/floe/log"
)

const rootPath = "/build/api"

// LaunchWeb sets up all the http routes runs the server and launches the trigger flows
func LaunchWeb(host string, hub *hub.Hub) {

	var rp = rootPath // just to shorten it ;)

	r := httprouter.New()
	r.HandleMethodNotAllowed = false
	r.NotFound = notFoundHandler{}
	r.PanicHandler = panicHandler

	println(len(hub.Config().Flows))

	h := handler{hub: hub}
	// --- authentication ---
	r.POST(rp+"/login", h.mw(loginHandler, false))
	r.POST(rp+"/logout", h.mw(logoutHandler, true))

	// --- api ---
	r.GET(rp+"/flows", h.mw(hndAllFlows, true))

	// --- CORS ---
	r.OPTIONS(rp+"/*all", h.mw(nil, false)) // catch all options

	// --- subscription endpoints ---
	h.setupSubs(rp+"/subs/", r, hub)
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
