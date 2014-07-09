package main

import (
	"net/http"
	"strings"
	"third_party/negroni"
	"time"
)

func runCommandLine() {
	exec("main launcher", 2*time.Second)
}

// serve as an rpc
func runAgent() {
}

func JsonHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS, DELETE")

	w.Header().Set("Access-Control-Allow-Headers", strings.Join(r.Header["Access-Control-Request-Headers"], ","))
}

func runWeb() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/exec", func(w http.ResponseWriter, req *http.Request) {
		JsonHeaders(w, req)
		exec("main launcher", 0)
	})

	mux.HandleFunc("/api/flow", func(w http.ResponseWriter, req *http.Request) {
		JsonHeaders(w, req)
		w.Write(project.ToJson())
	})

	n := negroni.Classic()

	n.UseHandler(mux)
	n.Run(":3000")
}

func main() {
	setup()

	//TODO mutex enum for mode on the commandline
	server := false
	agent := false

	if server {
		runWeb()
	} else if agent {
		runAgent()
	} else {
		runCommandLine()
	}
}
