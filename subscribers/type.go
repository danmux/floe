package subscribers

import (
	"github.com/floeit/floe/event"
	"github.com/julienschmidt/httprouter"
)

type Subscriber interface {
	PostHandler(queue *event.Queue) httprouter.Handle
	GetHandler(queue *event.Queue) httprouter.Handle
}

// Subs is a map pf all subscription types. This map will be used to attach
// Subscribers to the http server
var Subs = map[string]Subscriber{
	"data": data{},
}
