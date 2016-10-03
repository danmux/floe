package floe

import (
	"time"

	"github.com/floeit/floe/log"
	"github.com/floeit/floe/workfloe/par"
)

// one trigger floe to each launcher. each trigger launcher may have many trigger nodes that could fire
// independently.
type TriggerLink struct {
	Trigger  *Launcher
	Launcher *Launcher
}

// Run starts this trigger floe in a continuous loop
func (tf *TriggerLink) Run() {
	// start looping round
	go func() {
		for {
			tf.inner()
			time.Sleep(time.Second * 5) // just to slow things down a bit if the trigger is badly behaved
		}
	}()
}

// inner will launch the trigger floe and
func (tf *TriggerLink) inner() {

	// the trigger floe has not only to be single threaded but also synchronous
	ec := make(chan *par.Params)

	go tf.Trigger.startTrigger(time.Second, ec)

	log.Info("started trigger floe waiting to trigger: ", tf.Trigger.id)

	res := <-ec

	log.Info("trigger floe starting a launcher", res)

	if res.Status != 0 {
		log.Info("TRIGGER FAILED")
		return
	}

	// TODO queue up the floes here - dedupe repetitions - distribute work to latent agents

	// did this trigger floe have another floe to trigger - it should do or else whats the point
	if tf.Launcher == nil {
		log.Info("What a dumb trigger link it has no launcher", tf.Trigger.id)
	}

	fc := make(chan *par.Params)

	go tf.Launcher.Start(time.Second, fc)

	log.Info("trigger:", tf.Trigger.id, "launched", tf.Launcher.id)

	// right now because we wait here for the end chanel this trigger can only launch one at a time
	res = <-fc

	log.Info("end trigger launched floe", res)

	if res.Status == 0 {
		log.Info("TRIGGERED FLOW SUCCEEDED")
	} else {
		log.Info("TRIGGERED FLOW FAILED")
	}
}

// triggerFloe satisfies LaunchableFloe so can be launched by the flowlauncher
// this is a basic trigger floe that is automatically linked to the launcher it triggers
// in AddLauncher
type TriggerFloe struct {
	BaseLaunchable
	Func WorkfloeFunc
}

// GetProps returns a set of properties fo this floe, this function helps satisfy LaunchableFloe
func (tf *TriggerFloe) GetProps() *par.Props {
	p := tf.DefaultProps()
	(*p)[par.KeyTidyDesk] = "keep" // this to not trash the workspace
	return p
}

func (tf *TriggerFloe) FloeFunc(threadID int) *Workfloe {
	return tf.Func(threadID)
}
