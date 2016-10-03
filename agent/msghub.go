package agent

import (
	"encoding/json"
	"time"

	"github.com/floeit/floe/log"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 30 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type sockHub map[*sock]*sock

var hub sockHub

// Write accepts a string and a plain interface and serialises the interface to the websocket
// this also satisfies the StatusObserver as needed by the Launcher
func (h sockHub) Write(t string, p interface{}) {
	pkt := &packet{
		Type:    t,
		Payload: p,
	}
	for _, s := range h {
		s.write(pkt)
	}
}

// handleWS - Every new web socket connection is handled by this
func handleWS(ws *websocket.Conn) {
	if hub == nil {
		hub = map[*sock]*sock{}
	}

	// add this socket to the hub
	s := newSock(ws)
	hub[s] = s

	defer s.Close()

	// set ping loop pinging
	go s.pingLoop()

	// and the writer loop
	go s.writerLoop()

	s.readerLoop()
}

// our main message hub object created for every new ws connection
type sock struct {
	ws         *websocket.Conn // the connection
	writeChan  chan packet     // all return messages are pushed to this - so we can serialise use of the socket
	pingTicker *time.Ticker    // pings the client
	tickCloser chan bool       // so we can stop the above
	closed     bool            // flag the hub as closed
}

func newSock(ws *websocket.Conn) *sock {
	// set up some ws constraints
	ws.SetReadLimit(maxMessageSize)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { // got a pong back
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// return the new hub
	return &sock{
		ws:         ws,
		writeChan:  make(chan packet),
		pingTicker: time.NewTicker(pingPeriod),
		closed:     false,
		tickCloser: make(chan bool),
	}
}

func (mw *sock) Close() {
	if !mw.closed {
		mw.closed = true
		mw.pingTicker.Stop()
		mw.tickCloser <- true
		close(mw.writeChan)
		mw.ws.Close()
		delete(hub, mw)
	}
}

// queue up messages on the output queue - or to the machine in batch mode
func (mw *sock) write(pkt *packet) {
	if mw.closed { // throw away any messages if we are shut for business
		return
	}
	// pipe it all on the out channel
	mw.writeChan <- *pkt
}

// serialise writes to the websocket by ranging over the writeChan
func (mw *sock) writerLoop() {
	for pkt := range mw.writeChan {
		mw.ws.SetWriteDeadline(time.Now().Add(writeWait))

		data, err := json.Marshal(pkt)
		if err != nil {
			continue
		}
		err = mw.ws.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("failed to write to socket", err)
		}
	}
}

// keep reading off the input to this channel - throttled to 10ps then 1ps
func (mw *sock) readerLoop() {

	// start reading
	for !mw.closed {
		_, b, err := mw.ws.ReadMessage() // get the next frame

		// check all error states
		if err != nil {
			// no error if connection closed
			if cerr, ok := err.(*websocket.CloseError); ok {
				log.Info("client closing socket", cerr.Code, cerr.Text) // info
				break
			}
			// get here if some none closed socket problem
			log.Error("websocket pkt error", err)
			break
		}

		// if we think things should be closed then ignore the error
		if mw.closed {
			log.Warning("hub closed but with more messages")
			break
		}

		// route this message to the right place
		go mw.routeMessage(b) // do something with it
	}
	log.Info("closing socket")
}

func (mw *sock) pingLoop() {
	for !mw.closed {
		select {
		case <-mw.pingTicker.C:
			if err := mw.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Warning("websocket ping error", err)
			}
		case <-mw.tickCloser:
			break
		}
	}
}

// deserialise the common head part and call message handler terminate - calls mw.Close
func (mw *sock) routeMessage(b []byte) {

	// TODO deserialise something...

	// do some stuff
	// switch dude {
	// case "thing":

	// default:

	// }

}

// the generic form of a response message
type packet struct {
	Type    string
	Payload interface{}
}
