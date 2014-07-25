package main

import (
	"time"
)

// command line with 2 second step delay
func runCommandLine() {
	exec("main launcher", 2*time.Second)
}

// serve as an rpc
func runAgent() {
}

func main() {
	setup()

	//TODO mutex enum for mode on the commandline
	server := true
	agent := false

	if server {
		runWeb()
	} else if agent {
		runAgent()
	} else {
		runCommandLine()
	}
}
