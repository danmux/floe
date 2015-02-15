package main

import (
	"encoding/json"
	"github.com/codegangsta/negroni"
	"net/http"
	"strings"
	"time"
)

const rootFolder = "/build"

type ExecInstruction struct {
	Id      string
	Command string
	Delay   time.Duration
}

type ConfigRequest struct {
	FlowId string
	NodeId string
}

// api/exec
func execHandler(w http.ResponseWriter, req *http.Request) {
	JsonHeaders(w, req)

	if req.Method == "POST" {
		v := ExecInstruction{
			Delay: 1,
		}

		err := decodeBody(req, &v)
		if err != nil {
			respondWithJson(w, http.StatusNotAcceptable, err.Error())
			return
		}

		v.Delay = v.Delay * time.Second

		_, err = exec_async(v.Id, v.Delay)

		if err != nil {
			respondWithJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJson(w, http.StatusOK, nil)

	} else {
		respondWithJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func taskNodeConfigHandler(w http.ResponseWriter, req *http.Request) {
	JsonHeaders(w, req)

	if req.Method == "POST" {
		v := ConfigRequest{}

		err := decodeBody(req, &v)
		if err != nil {
			respondWithJson(w, http.StatusNotAcceptable, err.Error())
			return
		}

		if err != nil {
			respondWithJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJson(w, http.StatusOK, nil)

	} else {
		respondWithJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func stopHandler(w http.ResponseWriter, req *http.Request) {
	JsonHeaders(w, req)

	if req.Method == "POST" {
		v := ExecInstruction{}

		err := decodeBody(req, &v)
		if err != nil {
			respondWithJson(w, http.StatusNotAcceptable, err.Error())
			return
		}

		err = stop(v.Id)

		if err != nil {
			respondWithJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJson(w, http.StatusOK, nil)

	} else {
		respondWithJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func curStatHandler(w http.ResponseWriter, req *http.Request) {
	JsonHeaders(w, req)

	if req.Method == "GET" {

		project.ColectResults()

		respondWithJson(w, http.StatusOK, project.LastResults)
	} else {
		respondWithJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func decodeBody(req *http.Request, v interface{}) error {
	defer req.Body.Close()

	dec := json.NewDecoder(req.Body)

	err := dec.Decode(v)

	return err
}

func JsonHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS, DELETE")

	w.Header().Set("Access-Control-Allow-Headers", strings.Join(r.Header["Access-Control-Request-Headers"], ","))
}

func respondWithJson(w http.ResponseWriter, code int, v interface{}) {
	b := []byte(`{"status": "ok"}`)
	var err error
	if v != nil {
		b, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	w.WriteHeader(code)
	w.Write(b)
}

func runWeb(host string) {
	mux := http.NewServeMux()

	mux.HandleFunc(rootFolder+"/api/exec", execHandler)
	mux.HandleFunc(rootFolder+"/api/status/current", curStatHandler)
	mux.HandleFunc(rootFolder+"/api/status/tasknode", taskNodeConfigHandler)
	mux.HandleFunc(rootFolder+"/api/stop", stopHandler)

	mux.HandleFunc(rootFolder+"/api/flow", func(w http.ResponseWriter, req *http.Request) {
		JsonHeaders(w, req)
		w.Write(project.ToJson())
	})

	n := negroni.Classic()
	// n := negroni.New()

	n.UseHandler(mux)
	n.Run(host)
}
