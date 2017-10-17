package push

import (
	"github.com/floeit/floe/event"
	"github.com/julienschmidt/httprouter"
)

// Push defines the http push handlers that can send events to a queue
// the handler funcs returns a handler closed on the event queue
type Push interface {
	PostHandler(queue *event.Queue) httprouter.Handle
	GetHandler(queue *event.Queue) httprouter.Handle
	RequiresAuth() bool // this trigger expects to be authenticated with the server
}
