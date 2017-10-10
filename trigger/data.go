package trigger

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

// data
type Data struct{}

func (d Data) RequiresAuth() bool {
	return true
}

func (d Data) PostHandler(queue *event.Queue) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, par httprouter.Params) {
		o := struct {
			Ref     config.FlowRef
			Answers nt.Opts
		}{}

		if !decodeJSONBody(w, req, &o) {
			return
		}

		// add a data event - including a specific targeted Run if given
		queue.Publish(event.Event{
			RunRef: &event.RunRef{
				FlowRef: o.Ref,
			},
			Tag:  "data",
			Opts: o.Answers,
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
		log.Info(err)
		log.Infof("%#v", pl)
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
