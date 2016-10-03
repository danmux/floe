/*
Package floe coordinates flows of tasks

Workfloe - the heart of the executable graph - is instantiated from the LaunchableFloe.FloeFunc() implementation.

Launchable - is the interface which if satisfied has the function FloeFunc which when called constructs a WorkFloe
                 normally customefloes implement this. An implementation of LaunchableFloe.GetProps returns
                 any initial properties for the Floe.

floeLauncher - is instantiated with a LaunchableFlow - it will assume the same name and id as the LaunchableFlow, and call
				FloeFunc as many times as threads specifies.
				It only retains a reference to the floefunc, and an instance of LaunchableFloe's initial Props which
				are copied per thread.

TriggerFloe - are stand alone classes that link a trigger and a launcher
*/
package floe
