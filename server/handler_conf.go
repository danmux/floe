package server

import (
	"net/http"

	"github.com/floeit/floe/client"
)

// hostConfig is the publishable config of a host
type hostConfig struct {
	HostID string
	Online bool
	Tags   []string
}

// the /config endpoint
func confHandler(rw http.ResponseWriter, r *http.Request, ctx *context) (int, string, renderable) {
	cnf := struct {
		Config   hostConfig
		AllHosts map[string]client.HostConfig
	}{
		Config: hostConfig{
			HostID: ctx.hub.HostID(),
			Online: true, // TODO consider the option to pretend to be offline
			Tags:   ctx.hub.Tags(),
		},
		AllHosts: ctx.hub.AllHosts(),
	}

	return rOK, "OK", cnf
}
