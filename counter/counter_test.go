package counter

import (
	"sync"
	"testing"
)

func TestSimple(t *testing.T) {
	New(GetSimpleInc())
	i, ok := Inc("test")
	if !ok {
		t.Error("not ok")
	}
	if i != 1 {
		t.Error("not 1")
	}

	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 10; i++ {
		go func() {
			_, iok := Inc("test")
			if !iok {
				t.Error("not ok")
			}
			wg.Done()
		}()
		go func() {
			_, iok := Inc("test2")
			if !iok {
				t.Error("not ok")
			}
			wg.Done()
		}()
	}
	wg.Wait()
	i, ok = Inc("test")
	if !ok {
		t.Error("not ok")
	}
	if i != 12 {
		t.Error("not 12")
	}
	i, ok = Inc("test2")
	if !ok {
		t.Error("not ok")
	}
	if i != 11 {
		t.Error("not 11")
	}
}
