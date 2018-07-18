package hub

import (
	"sync"
	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/log"
)

type timerTrigger func(*Hub, *timer)

type timer struct {
	flow    config.FlowRef
	nodeID  string
	period  int // time between triggers in seconds
	next    time.Time
	trigger timerTrigger // the function to fire
}

type timers struct {
	mu   sync.RWMutex
	list map[string]*timer
}

func newTimers(h *Hub) *timers {
	t := &timers{
		list: map[string]*timer{},
	}

	go func() {
		for now := range time.Tick(time.Second) {
			t.mu.RLock()
			for name, tim := range t.list {
				if !now.After(tim.next) {
					continue
				}
				tim.next = now.Add(time.Duration(tim.period) * time.Second)

				log.Debugf("<%s> - timer trigger", name)
				tim.trigger(h, tim)
			}
			t.mu.RUnlock()
		}
	}()
	return t
}

func (t *timers) register(flow config.FlowRef, nodeID string, opts nt.Opts, trigger timerTrigger) {
	period, ok := opts["period"].(int)
	if !ok {
		period = 10
	}
	t.mu.Lock()
	t.list[flow.String()+"-"+nodeID] = &timer{
		flow:    flow,
		nodeID:  nodeID,
		period:  period,
		next:    time.Now().UTC().Add(time.Duration(period) * time.Second),
		trigger: trigger,
	}
	t.mu.Unlock()
}

func startFlowTrigger(h *Hub, tim *timer) {
	// set up the info needed to identify the trigger
	source := config.NodeRef{
		Class: "trigger",
		ID:    tim.nodeID,
	}
	opts := nt.Opts{
		"period": tim.period,
	}

	flow := h.config.Flow(tim.flow)
	if flow == nil {
		log.Errorf("<%s> - timer trigger no longer has a flow in config", source)
		return
	}

	ref, err := h.addToPending(flow, h.hostID, source, opts)
	if err != nil {
		log.Errorf("<%s> - from timer trigger did not add to pending: %s", source, err)
		return
	}
	log.Debugf("<%s> - from timer trigger added to pending", ref)
}
