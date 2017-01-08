package event

import "testing"

type listener struct {
	what func(e Event)
}

func (l *listener) Notify(e Event) {
	l.what(e)
}

func TestQueue(t *testing.T) {
	q := Queue{}
	fired := false
	done := make(chan bool)
	l := &listener{
		what: func(e Event) {
			fired = true
			if e.ID != 1 {
				t.Error("event ID wrong", e.ID)
			}
			if e.RunRef.Run.String() != "h1-12" {
				t.Error("bad hosted ref", e.RunRef.Run.String())
			}
			done <- true
		},
	}

	q.Register(l)

	e := Event{
		RunRef: &RunRef{
			Run: HostedIDRef{
				HostID: "h1",
				ID:     12,
			},
		},
	}

	q.Publish(e)
	<-done
	if !fired {
		t.Error("notify not fired")
	}
}
