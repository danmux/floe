package push

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

// Data is the push data endpoint handler
type Data struct{}

// RequiresAuth - decides if it needs a token.
func (d Data) RequiresAuth() bool {
	return true
}

// PostHandler handles POST requests
func (d Data) PostHandler(queue *event.Queue) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, par httprouter.Params) {
		log.Debug("got data push request")
		type form struct {
			ID     string
			Values nt.Opts
		}
		o := struct {
			Ref  config.FlowRef
			Run  string
			Form form
		}{}

		if !decodeJSONBody(w, req, &o) {
			return
		}

		/*
			{
			   "Ref":{
			      "ID":"build-project",
			      "Ver":1
			   },
			   "Run":"h1-7",
			   "Form":{
			      "ID":"sign-off",
			      "Values":{
			         "tests_passed":"rewqrw",
			         "to_hash":"rewqre"
			      }
			   }
			}
		*/

		rr := event.RunRef{
			FlowRef: o.Ref,
		}

		sourceNode := config.NodeRef{
			Class: "exec",
			ID:    o.Form.ID,
		}

		// if a run is given then it is data targetting a data input node
		if o.Run != "" {
			ps := strings.Split(o.Run, "-")
			if len(ps) == 2 {
				id, err := strconv.ParseInt(ps[1], 10, 64)
				if err != nil {
					log.Error("could not parse run id", err)
				} else {
					rr.Run.HostID = ps[0]
					rr.Run.ID = id
				}
			}
		}

		// add a data event - including a specific targeted Run if given
		queue.Publish(event.Event{
			RunRef:     rr,
			Tag:        "inbound.data",
			SourceNode: sourceNode,
			Opts:       o.Form.Values,
		})

		jsonResp(w, http.StatusOK, "OK", nil)
	}
}

func (d Data) GetHandler(queue *event.Queue) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, par httprouter.Params) {
		jsonResp(w, http.StatusOK, "OK", nil)
	}
}

func jsonResp(w http.ResponseWriter, code int, msg string, pl interface{}) {
	r := struct {
		Message string
		Payload interface{}
	}{
		Message: msg,
		Payload: pl,
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Debug(err)
		log.Debugf("%#v", pl)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"Message": "marshal failed", "Payload": "` + err.Error() + `"}`))
		return
	}

	w.WriteHeader(code)
	w.Write(b)
}

func decodeJSONBody(rw http.ResponseWriter, r *http.Request, v interface{}) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		jsonResp(rw, http.StatusBadRequest, "decoding json failed", err.Error())
		return false
	}
	return true
}
