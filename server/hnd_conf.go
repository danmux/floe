package server

import (
	"net/http"

	"github.com/floeit/floe/client"
)

// HostConfig is the publishable config
type HostConfig struct {
	HostID string
	Online bool
	Tags   []string
}

func confHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	cnf := struct {
		Config   HostConfig
		AllHosts map[string]client.HostConfig
	}{
		Config: HostConfig{
			HostID: ctx.hub.HostID(),
			Online: true, // TODO consider the option to pretend to be offline
			Tags:   ctx.hub.Tags(),
		},
		AllHosts: ctx.hub.AllHosts(),
	}

	return rOK, "OK", cnf
}
