package flow

import (
	"encoding/json"
	"github.com/golang/glog"
	"time"
)

type TriggerFlow struct {
	trigger  *FlowLauncher
	launcher *FlowLauncher
}

func (tf *TriggerFlow) Run() {

	// start looping round
	go func() {
		for {
			tf.inner()
		}
	}()
}

func (tf *TriggerFlow) inner() {

	// the trigger flow has not only to be single threaded but also synchronous
	ec := make(chan *Params)

	go tf.trigger.StartTrigger(time.Second, ec)

	glog.Infoln("started trigger:", tf.trigger.Id)

	res := <-ec

	glog.Infoln("end trigger", res)

	if res.Status != 0 {
		glog.Infoln("TRIGGER FAILED")
		return
	}

	// did this trigger flow have another flow to trigger
	if tf.launcher != nil {

		fc := make(chan *Params)

		go tf.launcher.Start(time.Second, fc)

		glog.Infoln("trigger:", tf.trigger.Id, "launched", tf.launcher.Id)

		res := <-fc

		glog.Infoln("end trigger launched flow", res)

		if res.Status == 0 {
			glog.Infoln("TRIGGERED FLOW SUCCEEDED")
		} else {
			glog.Infoln("TRIGGERED FLOW FAILED")
		}
	}

}

type triggerFlow struct {
	BaseLaunchable
}

func (l *triggerFlow) GetProps() *Props {
	p := l.DefaultProps()
	(*p)[KEY_TIDY_DESK] = "keep" // this to not trash the workspace
	return p
}

// the top level data structure that drives everytihng - contains many flows
// TODO - construct a rule structure that allows a succesfull workflow completion to trigger another workflow
type Project struct {
	Name          string
	FlowLaunchers map[string]*FlowLauncher
	LastResults   map[string]*FlowLaunchResult // a set of response stats by task id in our workflow for the last run
	RunList       map[string]*RunList          // historical set of run tasks
	Triggers      map[string]*TriggerFlow      // all the trigger flows
}

func MakeProject(name string) *Project {

	return &Project{
		Name:          name,
		FlowLaunchers: map[string]*FlowLauncher{},
		LastResults:   map[string]*FlowLaunchResult{},
		Triggers:      map[string]*TriggerFlow{},
	}
}

func (p *Project) MakeTriggerLauncher(name string, flowFunc GetFlowFunc) *FlowLauncher {
	triggerFlow := &triggerFlow{}

	triggerFlow.Init(name)

	launcher := &FlowLauncher{
		Props:    triggerFlow.GetProps(),
		Name:     triggerFlow.Name(),
		Id:       triggerFlow.Id(),
		flowFunc: flowFunc,
		Threads:  1,
	}
	launcher.sampleFlow = launcher.MakeFlow(0)
	p.AddFlow(launcher)

	return launcher
}

// attach to end events of the flows that get triggered so we know to trigger them again
// then launch all trigger flows
// at end of each trigger - launch the real flow
func (p *Project) RunTriggers() {
	for _, t := range p.Triggers {
		t.Run()
	}
}

func (p *Project) AddOrderedFlow(f *FlowLauncher, order int) {
	f.Order = order
	p.addFlow(f)
}

func (p *Project) AddFlow(f *FlowLauncher) {
	f.Order = len(p.FlowLaunchers)
	p.addFlow(f)
}

func (p *Project) addFlow(f *FlowLauncher) {
	pid := MakeID(f.Name)
	p.FlowLaunchers[pid] = f
	if f.trigger != nil {
		p.Triggers[MakeID(f.trigger.Name)] = &TriggerFlow{
			trigger:  f.trigger,
			launcher: f,
		}
	}
}

func (p *Project) AddTriggerFlow(triggerFlow *FlowLauncher) {

	glog.Info("add trigger:", triggerFlow.Name)

	triggerFlow.Order = len(p.Triggers)
	pid := MakeID(triggerFlow.Name)
	p.Triggers[pid] = &TriggerFlow{
		trigger:  triggerFlow,
		launcher: nil,
	}
}

func (p *Project) ColectResults() {
	for fid, flo := range p.FlowLaunchers {
		if flo.LastRunResult != nil {
			flo.LastRunResult.FlowId = fid
		}
		p.LastResults[fid] = flo.LastRunResult
	}
}

// a project structure is just a list of flowstructs so we can render the project graph as json
type projectStruct struct {
	Flows []FlowStruct
}

func (p Project) ToJson() []byte {
	ps := projectStruct{
		Flows: []FlowStruct{},
	}

	for _, f := range p.FlowLaunchers {
		ps.Flows = append(ps.Flows, f.GetStructure())
	}

	sJson, err := json.MarshalIndent(&ps, "", "  ")
	if err != nil {
		return nil
	}
	return sJson
}
