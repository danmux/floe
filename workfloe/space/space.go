// Package space manages all local file resources for a floe
package space

import (
	"path/filepath"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/task"
	"github.com/floeit/floe/workfloe/store"
)

// Conf a struct that holds information about the floe configuration and that can be referenced by the tasks to get at some floe properties
type Conf struct {
	id           string // the id of the launcher
	root         string // the root folder for all spaces
	TriggerStore store.Store
	HistoryStore store.Store
}

// NewConf creates a Floe configuration to set the executing environment for a floe
func NewConf(id, root string) *Conf {
	c := &Conf{
		id:   id,
		root: root,
	}

	ts, err := store.NewLocalStore(c.TriggerDataPath(), store.JSONMarshaler{})
	if err != nil {
		log.Fatal("cant make trigger store", c.TriggerDataPath(), err)
	}

	hs, err := store.NewLocalStore(c.HistoryDataPath(), store.JSONMarshaler{})
	if err != nil {
		log.Fatal("cant make history store", c.HistoryDataPath(), err)
	}
	c.TriggerStore = ts
	c.HistoryStore = hs
	return c
}

// WorkspacePath returns the full file path to the workspace
func (c *Conf) WorkspacePath() string {
	return filepath.Join(c.root, "wrk", c.id)
}

const datDir = "dat"

// TriggerDataPath returns the local file system path to this configs trigger directory
func (c *Conf) TriggerDataPath() string {
	return filepath.Join(c.root, datDir, "trig", c.id)
}

// HistoryDataPath returns the directory path of this configs history folder
func (c *Conf) HistoryDataPath() string {
	return filepath.Join(c.root, datDir, "hist", c.id)
}

// Context retuns a context to pass to the tasks
func (c *Conf) Context() *task.Context {
	return &task.Context{
		WorkspacePath:   c.WorkspacePath(),
		TriggerDataPath: c.TriggerDataPath(),
	}
}
