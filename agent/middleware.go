package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/floeit/floe/log"
	"github.com/julienschmidt/httprouter"
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

type renderable interface{}

func decodeBody(rw http.ResponseWriter, r *http.Request, v interface{}) bool {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		jsonResp(rw, http.StatusNotAcceptable, err.Error())
		return false
	}

	return true
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
	ps    *httprouter.Params
	agent *Agent
	sesh  *session
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

func (a *Agent) mw(f contextFunc, auth bool) func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Infof("RQ: %s %s", r.Method, r.URL.String())
		defer func() {
			log.Infof("RD: %s %s", r.Method, r.URL.String())
		}()

		cors(rw, r)

		// catch all with no handler
		if f == nil {
			jsonResp(rw, http.StatusOK, "good")
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
					log.Warning("cookie problem ", err)
				} else {
					tok = c.Value
				}
			}

			if tok == "" {
				jsonResp(rw, rUnauth, wrapper{Message: "missing session"})
				return
			}

			log.Info("checking token ", tok)

			// default to this agent for testing admin token
			if tok == a.ref.AdminToken {
				log.Info("found admin token ", tok)
				sesh = &session{
					token:      tok,
					lastActive: time.Now(),
					user:       "Admin",
				}
			}

			if sesh == nil {
				sesh = goodToken(tok)
				if sesh == nil {
					jsonResp(rw, rUnauth, wrapper{Message: "invalid session"})
					return
				}
			}

			// refresh cookie
			setCookie(rw, tok)
		}

		// got here then we are authenticated
		ctx := &context{
			ps:    &ps,
			sesh:  sesh,
			agent: thisAgent,
		}

		code, msg, res := f(rw, r, ctx)
		if code != 0 { // code 0 means a reply has been done somewhere else
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
