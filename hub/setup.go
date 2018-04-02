package hub

import (
	"os"
	"path/filepath"

	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
	"github.com/floeit/floe/path"
)

// enforceWS make sure there is a matching file system location and returns the workspace object
// shared will use the 'single' workspace
func (h Hub) enforceWS(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	ws, err := h.getWorkspace(runRef, single)
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

// getWorkspace returns the appropriate Workspace struct for this flow
func (h Hub) getWorkspace(runRef event.RunRef, single bool) (*nt.Workspace, error) {
	ebp, err := path.Expand(h.basePath)
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
