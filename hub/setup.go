package hub

import (
	"os"
	"path/filepath"

	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
)

// enforceWS make sure there is a matching file system location and returns the workspace object
// shared will use the 'single' workspace
func (h *Hub) enforceWS(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	ws, err := h.getWorkspace(runRef, single)
	if err != nil {
		return nil, err
	}
	err = os.RemoveAll(ws.BasePath)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(ws.BasePath, 0700)
	return ws, err
}

// getWorkspace returns the appropriate Workspace struct for this flow
func (h *Hub) getWorkspace(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	path := filepath.Join(h.basePath, "spaces", runRef.FlowRef.ID)
	if single {
		path = filepath.Join(path, "ws", "single")
	} else {
		path = filepath.Join(path, "ws", runRef.Run.String())
	}
	// setup the workspace config
	return &nt.Workspace{
		BasePath:   path,
		FetchCache: h.cachePath,
	}, nil
}
