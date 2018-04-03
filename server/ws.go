package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/websocket"

	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

type wsHub struct {
	sync.RWMutex
	cons map[*websocket.Conn]bool
}

func newWsHub() *wsHub {
	return &wsHub{
		cons: map[*websocket.Conn]bool{},
	}
}

func (w *wsHub) Notify(e event.Event) {
	w.RLock()
	defer w.RUnlock()

	b, err := json.Marshal(e)
	if err != nil {
		log.Error("json encoding event failed:", err)
		return
	}

	for ws := range w.cons {
		m, err := ws.Write(b)
		if err != nil {
			log.Fatal(err)
		}
		if m != len(b) {
			log.Errorf("ws write did not send full event (%d out of %d):", m, len(b))
		}
	}
}

func (w *wsHub) add(ws *websocket.Conn) {
	w.Lock()
	defer w.Unlock()

	log.Debug("ws - adding new client")

	w.cons[ws] = true
}

func (w *wsHub) remove(ws *websocket.Conn) {
	w.Lock()
	defer w.Unlock()

	delete(w.cons, ws)
}

func (w *wsHub) getWsHandler(h *handler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		sesh := authRequest(rw, r)
		if sesh == nil {
			return
		}
		h := websocket.Handler(w.handler)
		h.ServeHTTP(rw, r)
	}
}

func (w *wsHub) handler(ws *websocket.Conn) {
	w.add(ws)
	defer func() {
		w.remove(ws)
	}()

	for {
		msg := make([]byte, 512)
		n, err := ws.Read(msg)
		if err != nil {
			// normal client close
			if err == io.EOF {
				log.Debug("websocket - client closed")
			} else {
				log.Error("websocket - got an error", err)
			}
			err = ws.Close()
			if err != nil {
				log.Error("websocket - close error", err)
			}
			return
		}

		fmt.Printf("TODO - something with this - Receive: %s\n", msg[:n])
	}
}
