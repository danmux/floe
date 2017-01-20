package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/floeit/floe/log"
)

// HostConfig the public config data of a host
type HostConfig struct {
	HostID  string
	BaseURL string
	Online  bool
	Tags    []string
}

// FloeHost provides methods to access a host api
type FloeHost struct {
	sync.RWMutex

	// Config is the public config
	config HostConfig

	token string
}

func (f *FloeHost) GetConfig() HostConfig {
	f.RLock()
	defer f.RUnlock()
	return f.config
}

// New returns a new FloeHost
func New(base, token string) *FloeHost {
	fh := &FloeHost{
		config: HostConfig{
			BaseURL: base,
		},
		token: token,
	}
	// start the ping heartbeat
	go fh.pinger()
	return fh
}

type wrap struct {
	Message string
	Payload interface{}
}

func (f *FloeHost) pinger() {
	tk := time.NewTicker(time.Second * 10)
	for range tk.C {
		baseURL := f.config.BaseURL
		conf, err := f.fetchConf()
		f.Lock()
		if conf.HostID == "" || err != nil {
			log.Error("cant get config from", f.config.BaseURL, err)
			f.config.Online = false
		} else {
			f.config = conf
			f.config.Online = true
			f.config.BaseURL = baseURL
		}
		f.Unlock()
	}
}

func (f *FloeHost) fetchConf() (HostConfig, error) {
	w := wrap{}
	c := struct {
		Config HostConfig
	}{}
	w.Payload = &c
	code, err := f.get("/config", &w)
	if err != nil {
		return c.Config, err
	}
	if code == http.StatusOK {
		return c.Config, nil
	}
	return c.Config, nil
}

func (f *FloeHost) get(path string, r interface{}) (int, error) {
	return f.req("GET", path, nil, r)
}

func (f *FloeHost) post(path string, q, r interface{}) (int, error) {
	return f.req("POST", path, q, r)
}

func (f *FloeHost) put(path string, q, r interface{}) (int, error) {
	return f.req("PUT", path, q, r)
}

func (f *FloeHost) req(method, spath string, rq, rp interface{}) (status int, err error) {
	f.RLock()
	path := f.config.BaseURL + spath
	f.RUnlock()

	var b []byte
	if rq != nil {
		b, err = json.Marshal(rq)
		if err != nil {
			return 0, err
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(b))
	if err != nil {
		return 0, err
	}

	// add the auth
	req.Header.Add("X-Floe-Auth", f.token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if rp != nil {
		err = json.Unmarshal(body, rp)
		if err != nil {
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, nil
}
