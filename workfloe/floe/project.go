package floe

import (
	"errors"
	"os/user"
	"strings"
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/hist"
	"github.com/floeit/floe/workfloe/par"
	"github.com/floeit/floe/workfloe/space"
)

// Project is the top level data structure that drives everything - contains many launchers
type Project struct {
	Name      string
	ID        string
	rootWs    string                  // where the workspace will be
	launchers map[string]*Launcher    // the main floes launchers
	triggers  map[string]*TriggerLink // all the triggers linking a trigger launcher with a main floeLauncher
}

// NewProject instantiates and returns a Project
func NewProject() *Project {
	return &Project{
		rootWs:    "",
		launchers: map[string]*Launcher{},
		triggers:  map[string]*TriggerLink{},
	}
}

// AddLauncher adds a launcher to the project that will
func (p *Project) AddLauncher(launchable Launchable, threads int, initial *Launcher, trigger WorkfloeFunc) *Launcher {
	l := newLauncher(launchable, threads)

	if trigger == nil {
		// TODO add the default ux trigger
	} else {
		// make our trigger launchable floe
		// TODO add the normal UX trigger
		tf := &TriggerFloe{
			Func: trigger,
		}
		tf.Init(launchable.Name() + " trigger")

		tfl := newLauncher(tf, 1)

		log.Info("add trigger:", tfl.name)

		tfl.order = len(p.triggers)
		tfl.conf = space.NewConf(tfl.id, p.rootWs)
		pid := MakeID(tfl.name)
		p.triggers[pid] = &TriggerLink{
			Trigger:  tfl,
			Launcher: nil,
		}

		// l.Trigger = tfl
	}

	l.initial = initial
	l.order = len(p.launchers)

	p.addLauncher(l)
	return l
}

// Start launches a flow as specified by the floeID
func (p *Project) Start(floeID string, delay time.Duration, endChan chan *par.Params, obs StatusObserver) (int, error) {

	if p.ID == "" {
		log.Error("cant start - project Id not set")
		return -1, errors.New("project Id not found")
	}

	launcher, ok := p.launchers[floeID]

	if !ok {
		log.Error("cant start - floe not found ", floeID)
		return -1, errors.New("floe not found")
	}

	if launcher.isRunning() {
		log.Error("cant start - floe already running ", floeID)
		return -1, errors.New("floe already running")
	}

	launcher.obs = obs // pass in any event observer

	log.Info("starting:", floeID)

	launcher.Start(delay, endChan)

	log.Info("started:", floeID)

	return launcher.lastRunResult.RunID, nil
}

// Stop any floe in progress
func (p *Project) Stop(floeID string) error {

	launcher, ok := p.launchers[floeID]

	if !ok {
		log.Error("cant stop - floe not found ", floeID)
		return errors.New("floe not found")
	}

	launcher.exterminateExterminate()

	return nil
}

// RunTriggers attach to end events of the floes that get triggered so we know to trigger them again
// then launch all trigger floes
// at end of each trigger - launch the real floe
func (p *Project) RunTriggers() {
	for _, t := range p.triggers {
		t.Run()
	}
}

// SetName sets the name for the project and uses that to derive a project ID
func (p *Project) SetName(n string) {
	p.Name = n
	p.ID = MakeID(n)
}

// SetRoot sets the path of the root directory for all persistance for the project
func (p *Project) SetRoot(w string) {
	log.Info("Setting root folder:", w)
	usr, _ := user.Current()
	hd := usr.HomeDir

	if len(w) < 2 {
		return
	}
	// Check in case of paths like "/something/~/something/"
	if w[:2] == "~/" {
		w = strings.Replace(w, "~", hd, 1)
	}

	log.Info("Root folder:", w)
	p.rootWs = w
}

func (p *Project) addLauncher(f *Launcher) {
	// make the configuration
	f.conf = space.NewConf(f.id, p.rootWs)

	pid := MakeID(f.name)
	p.launchers[pid] = f
	// load in any history summaries
	f.runList = &hist.RunList{}
	f.runList.Load(f.conf.HistoryStore)
}

// Overview is the brief overview of the floe
type Overview struct {
	ID     string
	Name   string
	Order  int
	Status string
}

// ProjectOverview is just a list of Overviews for each floe in the project so we can render the project as json
type ProjectOverview struct {
	Name  string
	ID    string
	Floes []Overview
}

// RenderableProject returns the proxy struct (that renders json nicely) containing info about all floes in the project
func (p Project) RenderableProject() ProjectOverview {
	ps := ProjectOverview{
		Name:  p.Name,
		ID:    p.ID,
		Floes: []Overview{},
	}

	for _, fl := range p.launchers {
		status := "unknown"
		if fl.lastRunResult != nil {
			status = "fail"
			if fl.lastRunResult.Error == "" {
				status = "pass"
			}
		}
		s := fl.getStructure()
		ps.Floes = append(ps.Floes, Overview{
			ID:     s.ID,
			Name:   s.Name,
			Order:  s.Order,
			Status: status,
		})
	}

	return ps
}

// SummaryStruct is returned as the renderable strcut for a floe
type SummaryStruct struct {
	Floe Overview
	Runs *hist.RunList
}

// RenderableFloe returns the proxy struct that renders json nicely for a single floe including the run history
func (p Project) RenderableFloe(floeID string) SummaryStruct {

	ps := SummaryStruct{}

	f := p.launchers[floeID]
	if f == nil {
		return ps
	}

	// render the run history summaries
	ps.Runs = f.runList

	s := f.getStructure()

	status := "unknown"
	if f.lastRunResult != nil {
		status = "fail"
		if f.lastRunResult.Error == "" {
			status = "pass"
		}
	}

	ps.Floe = Overview{
		ID:     s.ID,
		Name:   s.Name,
		Order:  s.Order,
		Status: status,
	}

	return ps
}
