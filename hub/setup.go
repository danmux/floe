package hub

import (
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	nt "github.com/floeit/floe/config/nodetype"
	"github.com/floeit/floe/event"
)

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
