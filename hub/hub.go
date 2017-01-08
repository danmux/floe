package hub

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"time"

	"github.com/floeit/floe/config"
	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/log"
	"github.com/floeit/floe/store"
)

// Hub links events to the config rules
type Hub struct {
	basePath string         // the configured basePath for the hub
	hostID   string         // the id fo this host
	config   *config.Config // the config rules
	store    store.Store    // the thing to persist any state
	queue    *event.Queue   // the event q to route all events
	// runs contains list of runs ongoing or the archive
	// this is the only ongoing changing state the hub manages
	runs *RunStore
}

// New creates a new hub with the given config
func New(host, basePath string, c *config.Config, s store.Store, q *event.Queue) *Hub {
	h := &Hub{
		hostID:   host,
		basePath: basePath,
		config:   c,
		store:    s,
		queue:    q,
		runs:     newRunStore(s),
	}
	// hub subscribes to its own queue
	h.queue.Register(h)
	// start checking the pending queue
	go h.servicePending()

	return h
}

// Config returns the config for this hub
func (h *Hub) Config() config.Config {
	return *h.config
}

func (h *Hub) servicePending() {
	for range time.Tick(time.Second) {
		err := h.dispatchPending()
		if err != nil {
			log.Error(err)
		}
	}
}

// Notify is called whenever an event is sent to the hub. It
// makes the hub an Observer
func (h *Hub) Notify(e event.Event) {
	if e.RunRef == nil {
		// this is a an event from an external listener, thereby initiating a new flows
		err := h.pendFlowFromSubEvent(e)
		if err != nil {
			log.Error(err)
		}
		return
	}
	// otherwise it is a flow specific event then dispatch it to any active flows
	h.dispatchActive(e)
}

func (h *Hub) dispatchPending() error {
	for _, p := range h.runs.Pending.Todos {
		flow, ok := h.config.FindFlow(p.Ref.FlowRef, p.InitiatingEvent.Tag, p.InitiatingEvent.Opts)
		if !ok {
			return fmt.Errorf("pending not found %s, %s", p.Ref.FlowRef, p.InitiatingEvent.Tag)
		}

		// TODO - decide on best host

		// TODO if it is this host then decide if it can be executed immediately or should be queued
		// based on resource conflicts

		// if we are the preferred host and there are no resource conflicts we are good to go
		// add the active flow
		h.executePending(p, flow)
	}
	return nil
}

// activateFromSubs will find any flows that match the event and activate them
func (h *Hub) executePending(todo *Todo, flow config.FoundFlow) error {
	log.Debugf("<%s> (%s) start %s", todo.Ref.FlowRef, todo.Ref.Run, todo.InitiatingEvent.Tag)

	// setup the workspace config
	_, err := h.enforceWS(todo.Ref, flow.ReuseSpace)
	if err != nil {
		return err
	}

	// add the active flow
	err = h.runs.activate(todo, h.hostID)
	if err != nil {
		return err
	}

	// and then emit the subs node events
	for _, n := range flow.Nodes {
		h.queue.Publish(event.Event{
			RunRef:     &todo.Ref,
			SourceNode: n.NodeRef(),
			Tag:        getTag(n, "good"), // all subs emit good events
			Opts:       todo.InitiatingEvent.Opts,
		})
	}

	return nil
}

// queueFlowFromSubEvent will find any flows that match the event and add them to the pending list
func (h *Hub) pendFlowFromSubEvent(e event.Event) error {
	found := h.config.FindFlowsBySubs(e.Tag, e.Opts)

	// map the node config to events
	for f := range found {
		id, err := h.runs.addToPending(f, h.hostID, e)
		if err != nil {
			return err
		}
		log.Debugf("<%s> (%s) subs  %s", f, id, e.Tag)
	}

	return nil
}

func (h *Hub) dispatchActive(e event.Event) {
	// for all active flows find ones that match
	_, r := h.runs.findActiveRun(e.RunRef.Run)
	// no matching active run
	if r == nil {
		return
	}
	ns := h.config.FindNodeInFlow(r.Ref.FlowRef, e.Tag)
	for _, n := range ns {
		switch n.Class() {
		case config.NcTask:
			if n.TypeOfNode() == "end" { // special task type end the run
				h.endRun(r, n.NodeRef(), e.Opts)
				return
			}
			// asynchronous execute
			go h.executeNode(&r.Ref, n, e)
		case config.NcMerge:
			h.mergeEvent(r, n, e)
		}
	}
}

func (h *Hub) endRun(run *Run, source config.NodeRef, opts nt.Opts) {
	log.Debugf("<%s> (%s) DONE", run.Ref.FlowRef, run.Ref.Run)
	h.runs.end(run)
	// publish specific end run event
	e := event.Event{
		RunRef:     &run.Ref,
		SourceNode: source,
		Tag:        "sys.end.all",
		Opts:       opts,
	}
	h.queue.Publish(e)
}

func (h *Hub) mergeEvent(run *Run, node config.Node, e event.Event) {
	log.Debugf("<%s> (%s) merge %s", run.Ref.FlowRef, run.Ref.Run, e.Tag)

	waitsDone, opts := h.runs.updateWithMergeEvent(run, node.NodeRef().ID, e.Tag, e.Opts)
	// save the activeRun
	h.runs.Active.Save(activeKey, h.runs.store)
	// is the merge satisfied
	if (node.TypeOfNode() == "any" && waitsDone > 0) ||
		(node.TypeOfNode() == "all" && waitsDone == node.Waits()) {

		e := event.Event{
			RunRef:     &run.Ref,
			SourceNode: node.NodeRef(),
			Tag:        getTag(node, "all"), // when merges fire they emit the all event
			Opts:       opts,
		}
		h.queue.Publish(e)
	}
}

func (h *Hub) executeNode(runRef *event.RunRef, node config.Node, e event.Event) {
	log.Debugf("<%s> (%s) exec  %s", runRef.FlowRef, runRef.Run, e.Tag)

	// setup the workspace config
	ws, err := h.enforceWS(*runRef, false)

	status, opts, err := node.Execute(*ws, e.Opts)
	if err != nil {
		log.Debugf("<%s> (%d) error %v", runRef.FlowRef.ID, runRef.Run, err)
		return
	}
	// based on the int dispatch resultant events
	// first the all event - we dispatch in all result cases
	ne := event.Event{
		RunRef:     runRef,
		SourceNode: node.NodeRef(),
		Tag:        getTag(node, "all"),
		Opts:       opts,
	}
	h.queue.Publish(ne)

	// dispatch good or bad event
	if node.IsStatusGood(status) {
		ne.Tag = getTag(node, "good")
	} else {
		ne.Tag = getTag(node, "bad")
	}
	h.queue.Publish(ne)

	// dispatch the status specific event
	e.Tag = getTag(node, strconv.Itoa(status))
	h.queue.Publish(e)

}

func expandPath(w string) (string, error) {
	// cant use root or v small paths
	if len(w) < 2 {
		return "", errors.New("path too short")
	}

	b := strings.Split(w, "/")
	r := ""
	if b[0] == "" {
		r = string(filepath.Separator)
	}

	usr, _ := user.Current()
	hd := usr.HomeDir

	// Check in case of paths like "/something/~/something/"
	if b[0] == "~" {
		if b[1] == "" { // disallow "~/"
			return "", errors.New("root of user folder not allowed")
		}
		b[0] = hd
	}
	// replace %tmp with a temp folder
	if b[0] == "%tmp" {
		tmp, err := ioutil.TempDir("", "floe")
		if err != nil {
			return "", err
		}
		b[0] = tmp
	}

	return r + filepath.Join(b...), nil
}

// enforceWS make sure there is a matching file system location and returns the workspace object
// shared will use the 'single' workspace
func (h Hub) enforceWS(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	ws, err := h.getWS(runRef, single)
	if err != nil {
		return nil, err
	}
	err = os.RemoveAll(ws.BasePath)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(ws.BasePath, 0755)
	return ws, err
}

// getWS returns the appropriate Workspace struct for this flow
func (h Hub) getWS(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	ebp, err := expandPath(h.basePath)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(ebp, runRef.FlowRef.ID)
	if single {
		path = filepath.Join(path, "ws", "single")
	} else {
		path = filepath.Join(path, "ws", runRef.Run.String())
	}
	// setup the workspace config
	return &nt.Workspace{
		BasePath: path,
	}, nil
}

func getTag(node config.Node, subTag string) string {
	return fmt.Sprintf("%s.%s.%s", node.Class(), node.NodeRef().ID, subTag)
}
