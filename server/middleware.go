package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/hub"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/subscribers"
)

const (
	rOK       = http.StatusOK
	rUnauth   = http.StatusUnauthorized
	rBad      = http.StatusBadRequest
	rNotFound = http.StatusNotFound
	rErr      = http.StatusInternalServerError
	rCreated  = http.StatusCreated

	cookieName = "floe-sesh"
)

// AdminToken a configurable admin token for this host
var AdminToken string

type renderable interface{}

func decodeBody(rw http.ResponseWriter, r *http.Request, v interface{}) (bool, int, string) {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return false, rBad, err.Error()
	}

	return true, 0, ""
}

func jsonResp(w http.ResponseWriter, code int, r interface{}) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Info(err)
		log.Infof("%#v", r)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"Status": "Fail", "Payload": "` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(code)
	w.Write(b)
}

type context struct {
	ps   *httprouter.Params
	sesh *session
	hub  *hub.Hub
}

type notFoundHandler struct{}

func (h notFoundHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	jsonResp(rw, rNotFound, wrapper{Message: "not found"})
}

type contextFunc func(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable)

type wrapper struct {
	Message string
	Payload renderable
}

type handler struct {
	hub *hub.Hub
}

func (h handler) mw(f contextFunc, auth bool) func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var code int
		start := time.Now()
		log.Infof("req: %s %s", r.Method, r.URL.String())
		defer func() {
			log.Infof("rsp: %v %s %d %s", time.Since(start), r.Method, code, r.URL.String())
		}()

		cors(rw, r)

		// handler nil is the options catch all
		if f == nil {
			code = rOK
			jsonResp(rw, code, "ok")
			return
		}

		// pass this agent into the context

		var sesh *session
		if auth {
			tok := r.Header.Get("X-Floe-Auth")
			if tok == "" {
				log.Info("checking cookie")
				c, err := r.Cookie(cookieName)
				if err != nil {
					log.Warning("cookie problem", err)
				} else {
					tok = c.Value
				}
			}

			if tok == "" {
				code = rUnauth
				jsonResp(rw, code, wrapper{Message: "missing session"})
				return
			}

			log.Info("checking token ", tok, AdminToken)

			// default to this agent for testing admin token
			if tok == AdminToken {
				log.Info("found admin token", tok)
				sesh = &session{
					token:      tok,
					lastActive: time.Now(),
					user:       "Admin",
				}
			}

			if sesh == nil {
				sesh = goodToken(tok)
				if sesh == nil {
					code = rUnauth
					jsonResp(rw, code, wrapper{Message: "invalid session"})
					return
				}
			}

			// refresh cookie
			setCookie(rw, tok)
		}

		// got here then we are authenticated - so call the specific handler
		ctx := &context{
			ps:   &ps,
			sesh: sesh,
			hub:  h.hub,
		}

		code, msg, res := f(rw, r, ctx)
		// code 0 means the function responded itself
		if code == 0 {
			return
		}

		if msg == "" && code == rOK {
			msg = "OK"
		}
		reply := wrapper{
			Message: msg,
			Payload: res,
		}

		jsonResp(rw, code, reply)
	}
}

// setupSubs goes through all the subscriber types to set up the associated routes
func (h handler) setupSubs(path string, r *httprouter.Router, hub *hub.Hub) {
	for k, t := range subscribers.Subs {
		authenticated := false
		// data from a form in the app
		if k == "data" {
			authenticated = true
		}
		// TODO consider parameterised paths
		g := t.GetHandler(hub.Queue())
		if g != nil {
			r.GET(path+k, h.mw(adaptSub(hub, g), authenticated))
		}
		p := t.PostHandler(hub.Queue())
		if p != nil {
			r.POST(path+k, h.mw(adaptSub(hub, p), authenticated))
		}
	}
}

func adaptSub(hub *hub.Hub, handle httprouter.Handle) contextFunc {
	return func(w http.ResponseWriter, req *http.Request, ctx *context) (int, string, renderable) {
		handle(w, req, *ctx.ps)
		return 0, "", nil // each subscriber handler is responsible for the response
	}
}

func setCookie(rw http.ResponseWriter, tok string) {
	expiration := time.Now().Add(seshLifetime)
	cookie := http.Cookie{Name: cookieName, Value: tok, Expires: expiration}
	http.SetCookie(rw, &cookie)
}

func cors(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers", strings.Join(r.Header["Access-Control-Request-Headers"], ","))
}

func panicHandler(rw http.ResponseWriter, r *http.Request, v interface{}) {
	log.Error("PANIC in ", r.URL.String())
	log.Error(v)

	stack := debug.Stack()

	jsonResp(rw, http.StatusInternalServerError, string(stack))

	// send it to stderr
	fmt.Fprintf(os.Stderr, string(stack))
	// this sends it to the client....
	// fmt.Fprintf(rw, f, err, )
}
