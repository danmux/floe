Floe
====

A workflow engine, well suited to long running business process execution, for example:

* Continuous delivery.
* Continuous integration.
* Customer onboarding.


Quickstart
----------
start two host processes:

1. floe -tags=linux,go,couch -admin=123456 -host=h1 -bind=127.0.0.1:8080

2. floe -tags=linux,go,couch -admin=123456 -host=h2 -bind=127.0.0.1:8090


web 
---

http://localhost:8080/app/dash


Floe Terminology 
----------------
Flows are coordinated by nodes issuing events to which other nodes `listen`.

`Host` - Any named running Floe service, could be many on single compute unit (vm, container), or one each.

`Flow` - A description of a set of Nodes linked by events. A specific instance of an flow is a `Run`

`Node` - A config item that can respond to and issue events. A Node is part of a flow. Certain Nodes can Execute actions.
The types of Node are. 

* `Triggers` - These are http or polling nodes that respond to http requests or changes in a polled entity.
* `Tasks` - Nodes that on responding to an event execute some action, and then emit an event.
* `Merges` - A node that waits for `all` or `one` of a number of events before issuing its event.

`Run` - A specific invocation of a flow, can be in one of three states Pending, Active, Archive.

`RunRef` - An 'adopted' RunRef is a globally unique compound reference that resolves to a specific Run.

`Hub`  - is the central routing object. It instantiates Runs and executes actions on Nodes in the Run based on its config from any events it observes on its queue.

`Event` - Events are issued after a node has completed its duties. Other nodes are configured to listen for events. Certain other events are emitted that announce other state changes. Events are propagated to any clients connected via web sockets.

`Queue` - The hub event queue is the central chanel for all events.

`RunStore` - the hub references a run store that can persist the following lists - representing the three states fo a run..
* Pending list - Runs waiting to be executed, a Run on this list is called a Pend.
* Active List - Runs that are executing, and will be matched to events with matching adopted RunRefs. 
* Archive List - Runs that are have finished executing.


Life cycle of a flow
--------------------
When a trigger event arrives on the queue that matches a flow, the event reference will be considered 'un-adopted' this means it has not got a full run reference. A pending run is created with a globally unique compound reference (now adopted) - this reference (and some other meta data) is added to the pending list of the host that adopted it as a 'Pend' - this may not be the host that executes the run later.

A background process tries to assign Pend's to any host where the HostTags match, and where there are no Runs already matching the ResourceTags asked for - this allows certain nodes to be assigned to certain Runs, and to serialise Runs that need exclusive access to any third party, or other shared resources.

Once a Pend has been dispatched for execution it is moved out of the adopting Pending list and into the Active List on the executing host.

When one of the end conditions for a Run is met the Run is moved out of the Active list and into the Archive list on the host that executed the Run.


Config
------

### Common Config

All config has a Common section which has the following top level config items:

* `hosts`       - []string - all other floe Hosts
* `base-url`    - string - the api base url,  in case hosting on a sub domain.
* `config-path` - string - is a path to the config which can be a path to a file in a git repo e.g. git@github.com:floeit/floe.git/build/FLOE.yaml
* `store-type`  - string - define which type of store to use - memory, local, ec2


### Flow Config

A flow has the following top level config items:

* `id` - string - url friendly ID - computed from the name if not given explicitly.
* `ver`- int    - Flow version, together with an ID form a global compound unique key.
* `name` - string - human friendly name for the flow - will show up in web interface.
* `reuse-space`	- bool - If true then will use the single workspace and will mutex with other instances of this Flow on the same host.
* `host-tags` - []string - Tags that must match the tags on the host, useful for assigning specific flows to specific hosts.
* `resource-tags` - []string - Tags that represent a set of shared resources that should not be accessed by two or more runs. So if any flow has an active run on a host then no other flow can launch a run if the flow has any tags matching the one running.


### Triggers

Triggers are the things that start a flow off there are a few types of trigger.

* `data` - Where a web request pushing data to the server may trigger a flow - for example the web interface uses this, to explicitly launch a run.
* `timer` - A flow can be triggered periodically - as a timer does not contain any repo version info this can only include git 

### Exec Nodes

The most common type of node - executes a command, e.g. runs a mke command or a bash script.

`ignore-fail` - Only ever send the good event, even if the node failed. Can't be used in conjunction with UseStatus.
`cmd` - Use this if you are running an executable that only depends on the binary
`shell` - Use this if you are running something that requires the shell, e.g. bash scripts.
`args` - An array of command line arguments - for simple arguments these can be included space delimited in the `cmd` or `shell` lines, if there are quote enclosed arguments then use this args array.