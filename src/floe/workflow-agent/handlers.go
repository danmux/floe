package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"third_party/negroni"
	"time"
)

type ExecInstruction struct {
	Id      string
	Command string
	Delay   time.Duration
}

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

		fmt.Println("V", v)

		flow, err := exec_async(v.Id, v.Delay)
		fmt.Println("flow", flow)

		if err != nil {
			fmt.Println("ERORRO", err)
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
			fmt.Println("error", err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	w.WriteHeader(code)
	w.Write(b)
}

func runWeb() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/exec", execHandler)
	mux.HandleFunc("/api/status/current", curStatHandler)

	// mux.HandleFunc("/api/exec", func(w http.ResponseWriter, req *http.Request) {
	// 	JsonHeaders(w, req)
	// 	exec("main launcher", 0)
	// })

	mux.HandleFunc("/api/flow", func(w http.ResponseWriter, req *http.Request) {
		JsonHeaders(w, req)
		w.Write(project.ToJson())
	})

	n := negroni.Classic()
	// n := negroni.New()

	n.UseHandler(mux)
	n.Run(":3000")
}
