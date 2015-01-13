package main

import (
	"customfloe"
	"flag"
	"fmt"
	"time"
)

// command line with 2 second step delay
func runCommandLine(id string) {
	exec(id, 1*time.Second)
}

// serve as an rpc
func runAgent() {
	fmt.Println(string(project.ToJson()))
}

func main() {
	env := flag.String("env", "local", "any environment flag that filters the presented flows")
	host := flag.String("host", ":3000", "the host to bind to")
	flowId := flag.String("exec", "", "the flow id to execture directly from the command line")

	flag.Parse()

	setup(*env, customfloe.GetFlows)

	if *flowId != "" {
		runCommandLine(*flowId)
		return
	}

	runWeb(*host)

	// } else if agent {
	// 	runAgent()
	// }
}
