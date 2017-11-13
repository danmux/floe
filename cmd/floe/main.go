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
	adminToken := flag.String("admin", "", "admin token")
	tags := flag.String("tags", "master", "host tags")

	flag.Parse()

	cfg, err := ioutil.ReadFile(*in)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// TODO - implement real store
	s := store.NewMemStore()

	log.Debug(start(*host, *tags, *root, *bind, *adminToken, cfg, s, nil))
}

func start(host, tags, root, bind, adminToken string, conf []byte, store store.Store, addr chan string) error {
	c, err := config.ParseYAML(conf)
	if err != nil {
		return err
	}
	q := &event.Queue{}
	hub := hub.New(host, tags, root, adminToken, c, store, q)
	server.AdminToken = adminToken

	server.LaunchWeb(bind, c.Common.BaseURL, hub, q, addr)
	return nil
}
