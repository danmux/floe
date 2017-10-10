package trigger

import (
	"github.com/floeit/floe/event"
	"github.com/julienschmidt/httprouter"
)

// Trigger defines the http triggers that can send events to a queue
// the handler funcs returns a handler closed on an event queue
type Trigger interface {
	PostHandler(queue *event.Queue) httprouter.Handle
	GetHandler(queue *event.Queue) httprouter.Handle
	RequiresAuth() bool // this trigger expects to be authenticated with the server
}
