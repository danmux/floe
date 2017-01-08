package main

import (
	"flag"

	"io/ioutil"

	"os"

	"github.com/floeit/floe/config"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/hub"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/server"
	"github.com/floeit/floe/store"
)

func main() {
	root := flag.String("root", "~/.flow", "the root folder for configs and workspaces")
	in := flag.String("in", "config.yml", "the host config yaml")
	host := flag.String("host", "h1", "a short host name to use in id creation and routing")
	bind := flag.String("bind", ":8080", "what to bind the server to")
	admin := flag.String("admin", "", "admin token")

	flag.Parse()

	cfg, err := ioutil.ReadFile(*in)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// TODO - implement real store
	s := store.NewMemStore()

	start(*host, *root, *bind, *admin, cfg, s)
}

func start(host, root, bind, admin string, conf []byte, store store.Store) {
	c, _ := config.ParseYAML(conf)
	q := &event.Queue{}
	hub := hub.New(host, root, c, store, q)
	server.AdminToken = admin
	server.LaunchWeb(bind, hub)
}
