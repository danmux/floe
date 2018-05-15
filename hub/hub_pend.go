package hub

import (
	"fmt"
	"strings"
	"time"

	"github.com/floeit/floe/client"
	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
)

// This file contains all functions that deal with events that may or may not
// be serviced on this host i.e. pending runs - it uses the hub client to ask
// other nodes in the cluster if they can take a pending run.

// serviceLists attempts to dispatch pending flows
// TODO and times outs any active flows that are past their deadline
func (h *Hub) serviceLists() {
	for range time.Tick(time.Second * 5) {
		err := h.distributeAllPending()
		if err != nil {
			log.Error(err)
		}
	}
}

// distributeAllPending loops through all pending runs assessing whether they can be run then distributes them.
func (h *Hub) distributeAllPending() error {
	for _, p := range h.runs.allPends() {
		log.Debugf("<%s> - pending - attempt dispatch", p)

		if len(h.hosts) == 0 {
			log.Debugf("<%s> - pending - no hosts configured running job locally", p)
			ok, err := h.ExecutePending(p)
			if err != nil {
				return err
			}
			if !ok {
				log.Debugf("<%s> - pending - could not run job locally yet", p)
			} else {
				log.Debugf("<%s> - pending - job started locally", p)
				if err := h.removePend(p); err != nil {
					log.Error("could not save pending removal", err)
				}
			}
			continue
		}

		// Find candidate hosts that have a superset of the tags for the pending flow
		candidates := []*client.FloeHost{}
		for _, host := range h.hosts {
			cfg := host.GetConfig()
			if cfg.HostID == "" {
				continue // we have not communicated with the other host yet
			}
			log.Debugf("<%s> - pending - testing host %s with host tags: %v", p, cfg.HostID, cfg.Tags)
			if cfg.TagsMatch(p.Flow.HostTags) {
				log.Debugf("<%s> - pending - found matching host %s with host tags: %v", p, cfg.HostID, cfg.Tags)
				candidates = append(candidates, host)
			}
		}

		log.Debugf("<%s> - pending - found %d candidate hosts", p, len(candidates))

		// attempt to send it to any of the candidates
		launched := false
		for _, host := range candidates {
			if host.AttemptExecute(p) {
				log.Debugf("<%s> - pending - executed on <%s>", p, host.GetConfig().HostID)
				// remove from our pending list
				if err := h.removePend(p); err != nil {
					log.Error("could not save pending removal", err)
				}
				launched = true
				break
			}
		}

		if !launched {
			log.Debugf("<%s> - pending - no available host yet", p)
		}

		// TODO check pending queue for any pending run that is over age and send alert
	}
	return nil
}

// pendFlowFromTrigger uses the subscription fired event e to put any flows on the pending queue
// for any matching triggers.
func (h *Hub) pendFlowFromTrigger(e event.Event) error {
	if !strings.HasPrefix(e.Tag, inboundPrefix) {
		return fmt.Errorf("event %s dispatched to triggers does not have inbound tag prefix", e.Tag)
	}
	triggerType := e.Tag[len(inboundPrefix)+1:]

	log.Debugf("attempt to trigger type:<%s> (specified flow: %v)", triggerType, e.RunRef.FlowRef)

	// find any Flows with subs matching this event
	foundFlows := h.config.FindFlowsByTriggers(triggerType, e.RunRef.FlowRef, e.Opts)
	if len(foundFlows) == 0 {
		log.Debugf("no matching flow for type:'%s' (specified flow: %v)", triggerType, e.RunRef.FlowRef)
		return nil
	}

	// add each flow to the pending list
	for _, ff := range foundFlows {
		// make sure the flow has loaded in any references
		if ff.FlowFile != "" {
			log.Debugf("<%s> - getting flow from file '%s'", ff.Ref, ff.FlowFile)
			// grab the ref opts as a string if one exists
			ref := ""
			if r, ok := e.Opts["ref"]; ok {
				ref, _ = r.(string)
			}
			err := ff.Load(h.cachePath, ref)
			if err != nil {
				log.Errorf("<%s> - could not load in the flow from FlowFile: '%s'", ff.Ref, ff.FlowFile)
				continue
			}
		}

		// add the flow to the pending list making note of the node and opts that triggered it
		ref, err := h.addToPending(ff.Flow, h.hostID, ff.Matched[0].Ref, e.Opts)
		if err != nil {
			return err
		}
		log.Debugf("<%s> - from trigger type '%s' added to pending", ref, triggerType)
	}
	return nil
}

// addToPending adds a flow to the list of pending runs and publishes appropriate system state change event.
func (h *Hub) addToPending(flow *config.Flow, hostID string, trig config.NodeRef, opts nt.Opts) (event.RunRef, error) {
	ref, err := h.runs.addToPending(flow, hostID, trig, opts)
	if err != nil {
		return ref, err
	}

	h.queue.Publish(event.Event{
		RunRef: ref,
		Tag:    tagStateChange,
		Opts: nt.Opts{
			"action": "add-pend",
		},
		Good: true,
	})

	return ref, nil
}

// removePend removes the pend from the pending list issuing system state change event.
// Any error returned will be in the persisting of the pending list.
func (h *Hub) removePend(pend Pend) error {
	ok, err := h.runs.removePend(pend)
	if err != nil {
		return err
	}
	// If this did remove it from the pending list then send the system event.
	// The activate event can be used to remove from front end lists, instead of this event,
	// however this event can be fired even if an activate has not been.
	if ok {
		h.queue.Publish(event.Event{
			RunRef: pend.Ref,
			Tag:    tagStateChange,
			Opts: nt.Opts{
				"action": "remove-pend",
			},
			Good: true,
		})
	}
	return nil
}
