package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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

	log.Debug(start(*host, *tags, *root, *bind, *adminToken, cfg, nil))
}

func start(host, tags, root, bind, adminToken string, conf []byte, addr chan string) error {

	c, err := config.ParseYAML(conf)
	if err != nil {
		return err
	}

	var s store.Store
	switch c.Common.StoreType {
	case "", "memory":
		s = store.NewMemStore()
	case "local":
		s, err = store.NewLocalStore(filepath.Join(root, "store"))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s is not a supported store", c.Common.StoreType)
	}
	// TODO - implement other stores e.g. s3

	q := &event.Queue{}
	hub := hub.New(host, tags, filepath.Join(root, "spaces"), adminToken, c, s, q)
	server.AdminToken = adminToken

	server.LaunchWeb(bind, c.Common.BaseURL, hub, q, addr)
	return nil
}
