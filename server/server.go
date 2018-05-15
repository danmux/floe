package server

import (
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/event"
	"github.com/floeit/floe/hub"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/server/push"
)

const rootPath = "/build/api"

type Conf struct {
	PubBind string
	PubCert string
	PubKey  string

	PrvBind string
	PrvCert string
	PrvKey  string
}

// LaunchWeb sets up all the http routes runs the server and launches the trigger flows
// rp is the root path. Returns the address it binds to.
func LaunchWeb(conf Conf, rp string, hub *hub.Hub, q *event.Queue, addrChan chan string, webDev bool) {
	if rp == "" {
		rp = rootPath
	}
	r := httprouter.New()
	r.HandleMethodNotAllowed = false
	r.NotFound = notFoundHandler{}
	r.PanicHandler = panicHandler

	h := handler{hub: hub}

	// --- authentication ---
	r.POST(rp+"/login", h.mw(loginHandler, false))
	r.POST(rp+"/logout", h.mw(logoutHandler, true))

	// --- api ---
	r.GET(rp+"/flows", h.mw(hndAllFlows, true))          // list all the flows configs
	r.GET(rp+"/flows/:id", h.mw(hndFlow, true))          // return highest version of the flow config and run summaries from the cluster
	r.GET(rp+"/flows/:id/runs/:rid", h.mw(hndRun, true)) // returns the identified run detail (may be on another host)

	// --- push endpoints ---
	h.setupPushes(rp+"/push/", r, hub)

	// --- p2p api ---
	r.POST(rp+"/p2p/flows/exec", h.mw(hndP2PExecFlow, true))    // internal api to pass a pending todo to activate it on this host
	r.GET(rp+"/p2p/flows/:id/runs", h.mw(hndP2PRuns, true))     // all summary runs from this host for this flow id
	r.GET(rp+"/p2p/flows/:id/runs/:rid", h.mw(hndP2PRun, true)) // detailed run info from this host for this flow id and run id
	r.GET(rp+"/p2p/config", h.mw(confHandler, true))            // return host config and what it knows about other hosts

	// --- static files for the spa ---
	if webDev { // local development mode
		serveFiles(r, "/static/*filepath", http.Dir("webapp"))
		r.GET("/app/*filepath", zipper(singleFile("webapp/index.html")))
	} else { // release mode
		serveFiles(r, "/static/*filepath", assetFS())
		r.GET("/app/*filepath", zipper(assetFile("webapp/index.html")))
	}

	// serveFiles(r, "/static/img/*filepath", http.Dir("webapp/img"))
	// serveFiles(r, "/static/js/*filepath", http.Dir("webapp/js"))
	// serveFiles(r, "/static/font/*filepath", http.Dir("webapp/font"))

	// ws endpoint
	wsh := newWsHub()
	q.Register(wsh)
	r.GET("/ws", wsh.getWsHandler(&h))

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
		serveFiles(r, "/build/css/*filepath", http.Dir("public/build/css"))
		serveFiles(r, "/build/fonts/*filepath", http.Dir("public/build/fonts"))
		serveFiles(r, "/build/img/*filepath", http.Dir("public/build/img"))
		serveFiles(r, "/build/js/*filepath", http.Dir("public/build/js"))

	*/

	// start the private server if one is configured differently to the public server
	if conf.PrvBind != conf.PubBind && conf.PrvBind != "" {
		log.Debug("private server listen on:", conf.PrvBind)
		go launch(conf.PrvBind, conf.PrvCert, conf.PrvKey, r, nil)
	}

	// start the public server
	log.Debug("pub server listen on:", conf.PubBind)
	launch(conf.PubBind, conf.PubCert, conf.PubKey, r, addrChan)
}

func launch(bind, cert, key string, r http.Handler, addrChan chan string) {
	log.Debug("attempting to listen on:", bind)

	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	address := listener.Addr().(*net.TCPAddr).String()

	// in separate go routine message the passed in chan with the server address
	if addrChan != nil {
		go func() {
			addrChan <- address
		}()
	}

	log.Debug("starting on:", address)

	if cert != "" {
		log.Debug("using https")
		log.Fatal(http.ServeTLS(listener, r, cert, key))
	} else {
		log.Debug("using http")
		log.Fatal(http.Serve(listener, r))
	}
}

func singleFile(path string) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(rw, r, path)
	}
}

func assetFile(path string) httprouter.Handle {
	b := MustAsset(path)
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write(b)
	}
}

// pushes is the map of all trigger types that can be triggered via the trigger endpoints.
// This map will be used to attach these pushes types to the http server.
// The key here will be used as the sub path to route to this trigger.
var pushes = map[string]push.Push{
	"data": push.Data{},
}
