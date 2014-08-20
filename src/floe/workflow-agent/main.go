package main

import (
	"fmt"
	"time"
)

// command line with 2 second step delay
func runCommandLine() {
	exec("main-flow", 1*time.Second)
}

// serve as an rpc
func runAgent() {
	fmt.Println(string(project.ToJson()))
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
