package agent

import (
	"net/http"
	"time"

	"github.com/floeit/floe/log"
	"github.com/julienschmidt/httprouter"
)

const rootPath = "/build/api"

// LaunchWeb sets up all the http routes runs the server and launches the trigger floes
func (a *Agent) LaunchWeb(host string) {

	// after 5 seconds run all the triggers
	go func() {
		time.Sleep(time.Second * 5)
		a.project.RunTriggers()
	}()

	var rp = rootPath // just to shorten it ;)

	r := httprouter.New()
	r.HandleMethodNotAllowed = false

	r.NotFound = notFoundHandler{}

	r.PanicHandler = panicHandler

	// --- authentication ---
	r.POST(rp+"/login", a.mw(loginHandler, false))
	r.POST(rp+"/logout", a.mw(logoutHandler, true))

	// --- new api ---
	r.GET(rp+"/floes", a.mw(allFloeHandler, true))
	r.GET(rp+"/floes/:flid", a.mw(floeHandler, true))
	r.POST(rp+"/floes/:flid/exec", a.mw(execHandler, true))
	r.POST(rp+"/floes/:flid/stop", a.mw(stopHandler, true))
	r.GET(rp+"/floes/:flid/run/:agentid/:runid", a.mw(runHandler, true)) // get the current progress of a run for an agent and run

	// --- web socket connection ---
	r.GET(rp+"/msg", wsHandler)

	// --- CORS ---
	r.OPTIONS(rp+"/*all", a.mw(nil, false)) // catch all options

	// --- the web page stuff ---
	r.GET("/build/", indexHandler)
	r.ServeFiles("/build/css/*filepath", http.Dir("public/build/css"))
	r.ServeFiles("/build/fonts/*filepath", http.Dir("public/build/fonts"))
	r.ServeFiles("/build/img/*filepath", http.Dir("public/build/img"))
	r.ServeFiles("/build/js/*filepath", http.Dir("public/build/js"))

	log.Info("agent server starting on:", host)
	log.Fatal(http.ListenAndServe(host, r))
}
