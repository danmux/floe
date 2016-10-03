package floe

import (
	"flag"
	"time"

	"github.com/floeit/floe/agent"
)

var (
	env        string
	root       string
	host       string
	floeID     string
	adminToken string
)

// Prep sets up flags for the agent
func Prep() {
	flag.StringVar(&env, "env", "test", "any environment flag that filters the presented floes")
	flag.StringVar(&root, "root", "~/floe", "the root of the workspace and data folders")
	flag.StringVar(&host, "host", ":3000", "the host to bind to")
	flag.StringVar(&floeID, "exec", "", "the floe id to execute directly from the command line")
	flag.StringVar(&adminToken, "token", "you-must-change-this", "an admin token to allow floe agents to chat")
}

// Start starts the web agent or runs the command line exec
func Start(agentName, agentDesc string, floeFunc agent.GetFloesFunc) {
	flag.Parse()

	a := agent.NewAgent(agentName, agentDesc)
	a.SetToken(adminToken)
	a.Setup(env, floeFunc, root)

	// if we did not ask to run a specific floe then set up a web server
	if floeID == "" {
		println("launching web interface on: ", host)
		a.LaunchWeb(host)
	} else {
		// otherwise run the specific floe
		a.Exec(floeID, 2*time.Second)
		return
	}
}
