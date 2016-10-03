package agent

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/testfloe"
)

const adminToken = "a-test-admin-token"

var (
	once sync.Once
	base string
)

func TestWebUnauth(t *testing.T) {
	setup(t)
	resp := &genResp{}

	// auth should fail with a missing session
	ms := "missing session"
	webGet(t, "", "/floes", resp, []int{401}) // unauth
	if resp.Message != ms {
		t.Errorf("should have got `%s` but got: `%s`", ms, resp.Message)
	}

	// auth should fail with an invalid session
	ms = "invalid session"
	webGet(t, "unauth-tok", "/floes", resp, []int{401})
	if resp.Message != ms {
		t.Errorf("should have got `i%s` but got: `%s`", ms, resp.Message)
	}

	// admin token should authenticate
	if !webGet(t, adminToken, "/floes", resp, []int{200}) {
		t.Errorf("admin should have got floes: %s", resp.Message)
	}

	// logged in user should be authenticated
	token := webLogin(t)
	if !webGet(t, token, "/floes", resp, []int{200}) { // authed
		t.Errorf("logged in user should have got floes: %s", resp.Message)
	}
}

func TestWebLaunch(t *testing.T) {
	setup(t)

	tok := webLogin(t)
	floes := &floesResp{}
	resp := &genResp{
		Payload: floes,
	}

	if ok := webGet(t, tok, "/floes", resp, []int{200}); !ok { // authed
		t.Error("getting floes failed")
	}

	flid := floes.Floes[0].ID

	if flid != "test-build" {
		t.Fatal("bad floe ID", flid)
	}

	time.Sleep(time.Second * 3)

	resp = &genResp{}
	if ok := webGet(t, tok, "/floes/"+flid, resp, []int{200}); !ok {
		t.Error("getting floe failed")
	}

	p, _ := json.MarshalIndent(resp, "", "  ")
	println(string(p))

	// TODO runs = nil

	// Execute the flow
	// resp = &genResp{}
	// if ok := webGet(t, tok, "/floes/"+flid+"/exec", resp, []int{200}); !ok {
	// 	t.Error("executing floe failed")
	// }
}

func setupWeb(t *testing.T) {
	log.SetLevel(8)

	basePath := "/build/api"
	addr := "127.0.0.1:3013"
	base = "http://" + addr + basePath

	a := NewAgent("a1", "test agent 1")
	a.Setup("test", testfloe.GetFloes, "~/tmp/floe1")
	a.SetToken(adminToken)
	go a.LaunchWeb(addr)

	println("exec")
	a.Exec("test-build", 200*time.Millisecond)

	// addr2 := "127.0.0.1:3014"

	// a2 := NewAgent("a2", "test agent 2")
	// a2.Setup("test", testfloe.GetFloes, "~/tmp/floe2")
	// a2.SetToken(adminToken)
	// go a2.LaunchWeb(addr2)

	good := waitAPIReady(t)
	if !good {
		t.Fatal("failed to wait or server to come up")
	}
}

func setup(t *testing.T) {
	once.Do(func() {
		setupWeb(t)
	})
}

type genResp struct {
	Message string
	Payload interface{}
}

type summaryStruct struct {
	ID     string
	Name   string
	Order  int
	Status string
}

type floesResp struct {
	Floes []summaryStruct
}

func webLogin(t *testing.T) string {
	resp := &genResp{}
	v := struct {
		User     string
		Password string
	}{
		User:     "admin",
		Password: "password",
	}

	pl := struct {
		User  string
		Role  string
		Token string
	}{}

	resp.Payload = &pl

	webPost(t, "", "/login", v, resp, []int{200}) // login
	if resp.Message != "OK" {
		t.Errorf("login should have got `OK` but got: `%s`", resp.Message)
	}

	if pl.Token == "" {
		t.Errorf("login should have got none empty token")
	}
	return pl.Token
}

// func webLogout(t *testing.T) {
// 	resp := &genResp{}
// 	v := struct {
// 		User     string
// 		Password string
// 	}{
// 		User:     "admin",
// 		Password: "password",
// 	}

// 	pl := struct {
// 		User  string
// 		Role  string
// 		Token string
// 	}{}

// 	resp.Payload = &pl

// 	webPost(t, "/logout", v, resp, []int{200}) // login
// 	if resp.Message != "OK" {
// 		t.Errorf("login should have got `OK` but got: `%s`", resp.Message)
// 	}

// 	if pl.Token != "" {
// 		t.Errorf("login should have got none empty token")
// 	}
// }

// --- helpers n stuff
func waitAPIReady(t *testing.T) bool {
	for n := 0; n < 10; n++ {
		good := webReq(t, "OPTIONS", "", "/", nil, nil, []int{200}, false)
		if good {
			return true
		}
		time.Sleep(time.Millisecond * 250)
	}
	return false
}

func webGet(t *testing.T, tok, path string, r interface{}, expected []int) bool {
	return webReq(t, "GET", tok, path, nil, r, expected, true)
}

func webPost(t *testing.T, tok, path string, q, r interface{}, expected []int) bool {
	return webReq(t, "POST", tok, path, q, r, expected, true)
}

func webPut(t *testing.T, tok, path string, q, r interface{}, expected []int) bool {
	return webReq(t, "PUT", tok, path, q, r, expected, true)
}

func webReq(t *testing.T, method, tok, spath string, rq, rp interface{}, expected []int, fail bool) bool {
	path := base + spath

	var b []byte
	if rq != nil {
		var err error
		b, err = json.Marshal(rq)
		if err != nil {
			t.Error("Can't marshal request", err)
			rp = nil
			return false
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(b))

	if err != nil {
		panic(err)
	}

	req.Header.Add("X-Floe-Auth", tok)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		if fail {
			t.Error("Get failed", err)
		}
		rp = nil
		return false
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if fail {
			t.Error("Read get response failed", err)
		}
		rp = nil
		return false
	}

	// t.Log(string(body))

	codeGood := false
	for _, c := range expected {
		if resp.StatusCode == c {
			codeGood = true
			break
		}
	}

	if !codeGood {
		if fail {
			t.Errorf("Bad response %d, wanted one of %v for [%s:%s]", resp.StatusCode, expected, method, spath)
		}
		rp = nil
	}

	if rp != nil {
		err = json.Unmarshal(body, rp)
		if err != nil {
			if fail {
				t.Error("Failed to unmarshal response", err)
			}
			rp = nil
			return false
		}
	}

	return codeGood
}
