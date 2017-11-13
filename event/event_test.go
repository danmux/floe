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
		RunRef: RunRef{
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


func TestIsSystem(t *testing.T) {
	fix := []struct{
		tag string
		is bool
	}{
		{"", false},
		{"fo", false},
		{"system.foo", false},
		{"sys.bar", true},
	}
	for i, f := range fix {
		e := Event{
			Tag: f.tag,
		}
		if e.IsSystem() != f.is {
			t.Errorf("%d - tag %s should have had IsSystem: %v", i, f.tag, f.is)
		}
	}	
}

// The following benchmarks illustrate the relative performances of 
// passing th reference by value or as a pointer.
// as of 2017/11 there is 1ns in it...
// BenchmarkRunRefPBR-8   	500000000	         3.15 ns/op
// BenchmarkRunRefPBV-8   	300000000	         4.32 ns/op
func BenchmarkRunRefPBP(b *testing.B) {

	notnop := func (r *RunRef) {
		r.ExecHost = "" // hopefully avoid any optimisers?
	}

	r := &RunRef{}
	for i := 0; i < b.N; i++ {
		notnop(r)
	}
}

func BenchmarkRunRefPBV(b *testing.B) {

	notnop := func (r RunRef) {
		r.ExecHost = "" // hopefully avoid any optimisers?
	}

	r := RunRef{}
	for i := 0; i < b.N; i++ {
		notnop(r)
	}
}