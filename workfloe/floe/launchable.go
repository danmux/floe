package floe

import (
	"path/filepath"

	"github.com/floeit/floe/workfloe/par"
)

// WorkfloeFunc defines the function signature of the WorkfloeFunc that returns an instance of a Workfloe
type WorkfloeFunc func(threadId int) *Workfloe

// Launchable defines a thing that can be launched by a launcher. Clients of this api can implement this interface often with the help of composition with a BaseLaunchable
type Launchable interface {
	// FloeFunc returns a single constructed instance of a workflow. FloeFunc will be called for each thread specified in the launcher
	FloeFunc(threadID int) *Workfloe
	// Name is the name of the launchable
	Name() string
	// ID is the id of the launchable and is normally derived from the name
	ID() string
	// GetProps returns any initial properties for the launcher from this launchable
	GetProps() *par.Props
}

// BaseLaunchable is a used to compose with others so that they satisfy a good proportion of the Launchable interface and provide some boilerplate
type BaseLaunchable struct {
	name string
	id   string
}

// Init sets up the minimal name and ID derived from the name
func (b *BaseLaunchable) Init(name string) {
	b.name = name
	b.id = MakeID(name)
}

// Name returns the name of this launchable floe
func (b *BaseLaunchable) Name() string {
	return b.name
}

// ID returns the ID of this launchable floe
func (b *BaseLaunchable) ID() string {
	return b.id
}

// DefaultProps returns a handy set of default properties that each launchable can add to
func (b *BaseLaunchable) DefaultProps() *par.Props {
	props := par.Props{}

	props["path"] = string(filepath.Separator)
	props[par.KeyTidyDesk] = "reset" // or keep

	return &props
}

// StatusObserver is the interface that must be satisfied by any observer to this Launcher
type StatusObserver interface {
	Write(t string, p interface{})
}
