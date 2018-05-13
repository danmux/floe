Floe
====

A workflow engine, well suited to long running business process execution, for example:

* Continuous delivery.
* Continuous integration.
* Customer onboarding.

Quick Start
-----------
Download or build from scratch `floe` executable.

Start two host processes:

1. floe -tags=linux,go,couch -admin=123456 -host_name=h1 -pub_bind=127.0.0.1:8080

2. floe -tags=linux,go,couch -admin=123456 -host_name=h2 -pub_bind=127.0.0.1:8090

These commands default to reading in a `default.yml`

web 
---

http://localhost:8080/app/dash


Floe Terminology 
----------------
Flows are coordinated by nodes issuing events to which other nodes `listen`.

`Host`    - Any named running `floe` service, it could be many such processes on single compute unit (vm, container), or one each.

`Flow`    - A description of a set of `nodes` linked by events. A specific instance of an flow is a `Run`

`Node`    - A config item that can respond to and issue events. A Node is part of a flow. Certain Nodes can Execute actions.

The types of Node are. 

* `Triggers` - These start a run for their flow, and are http or polling nodes that respond to http requests or changes in a polled entity.
* `Tasks` - Nodes that on responding to an event execute some action, and then emit an event.
* `Merges` - A node that waits for `all` or `one` of a number of events before issuing its event.

`Run`      - A specific invocation of a flow, can be in one of three states Pending, Active, Archive.

`Workspace`- A place on disc for this run where most of the run actions should take place. Since flow can execute arbitrary scripts, there is no guarantee that mutations to storage are constrained to this workspace. It is up to the script author to isolate (e.g. using containers, or great care!) 

`RunRef`   - An 'adopted' RunRef is a globally unique compound reference that resolves to a specific Run.

`Hub`      - Is the central routing object. It instantiates Runs and executes actions on Nodes in the Run based on its config from any events it observes on its queue.

`Event`    - Events are issued after a node has completed its duties. Other nodes are configured to listen for events. Certain other events are emitted that announce other state changes. Events are propagated to any clients connected via web sockets.

`Queue`    - The hub event queue is the central chanel for all events.

`RunStore` - the hub references a run store that can persist the following lists - representing the three states of a run..
* `Pending` - Runs waiting to be executed, a Run on this list is called a Pend.
* `Active` - Runs that are executing, and will be matched to events with matching adopted RunRefs. 
* `Archive` - Runs that are have finished executing.

Life cycle of a flow
--------------------
When a trigger event arrives on the queue that matches a flow, the event reference will be considered 'un-adopted' this means it has not got a full run reference. A pending run is created with a globally unique compound reference (now adopted) - this reference (and some other meta data) is added to the pending list of the host that adopted it as a 'Pend' - this may not be the host that executes the run later. (These were called TODO's but that has a very particular meaning!)

A background process tries to assign Pend's to any host where the `HostTags` match, and where there are no Runs already matching the `ResourceTags` asked for - this allows certain nodes to be assigned to certain Runs, and to serialise Runs that need exclusive access to any third party, or other shared resources.

Once a Pend has been dispatched for execution it is moved out of the adopting Pending list and into the Active List on the executing host.

When one of the end conditions for a Run is met the Run is moved out of the Active list and into the Archive list on the host that executed the Run.

All of this is dealt with in the `Hub` the files are divided into three:

* `hub_setup.go` - The Hub definition and initial setup code.
* `hub_pend.go` - Code that handle events that trigger a pending run, and dispatches them to available hosts.
* `hub_exec.go` - Code that accepts a pending run and activates it, directs events to task nodes, and Executes tasks.

Config
------

### Common Config

All config has a Common section which has the following top level config items:

* `hosts`       - []string - all other floe Hosts
* `base-url`    - string - the api base url,  in case hosting on a sub domain.
* `config-path` - string - is a path to the config which can be a path to a file in a git repo e.g. git@github.com:floeit/floe.git/build/FLOE.yaml
* `store-type`  - string - define which type of store to use - memory, local, ec2


### Flow Config

**A note on the workspace var**
Many field values will expand to include the workspace.  Any `{{ws}}` will be replaced by the absolute workspace path.
Task fields that start `./` (and are not `./...`) will also be replaced by the absolute workspace path, as will any value `.` on its own.

A flow has the following top level config items:

* `id` - string - url friendly ID - computed from the name if not given explicitly.
* `ver`- int    - Flow version, together with an ID form a global compound unique key.
* `name` - string - human friendly name for the flow - will show up in web interface.
* `reuse-space`	- bool - If true then will use the single workspace and will mutex with other instances of this Flow on the same host.
* `host-tags` - ([]string) - Tags that must match the tags on the host, useful for assigning specific flows to specific hosts.
* `resource-tags` - ([]string) - Tags that represent a set of shared resources that should not be accessed by two or more runs. So if any flow has an active run on a host then no other flow can launch a run if the flow has any tags matching the one running.
* `env`     - ([]string) - In the form of key=value environment variable to be set in the context of the command being executed, can include `{{ws}}` to expand to full absolute path - `.` at the start will be treated like `{{ws}}`.

* `flow-file` - string - the reference to a file that can be loaded as the pending run is generated, this file will override the config of the floe - so can be used like a jenkinsfile, three types of reference can be used...
    * `file` - load it from the local file system. e.g. `floes/floe.yaml`
    * `git` - do the shallowest clone of the repo specified and grab the content e.g. `git@github.com:floeit/floe.git/build/FLOE.yaml` in this case if the opts contain a ref then the ref (git ref - e.g. tag, branch etc.) will be used
    *  `fetch` - Fetch a file via http(s) e.g. `https://raw.githubusercontent.com/floeit/floe/redesign/confog.yaml`

### Triggers

Triggers are the things that start a flow off there are a few types of trigger.

* `data` - Where a web request pushing data to the server may trigger a flow - for example the web interface uses this, to explicitly launch a run.
* `timer` - A flow can be triggered periodically - as a timer does not contain any repo version info this can only include git 

### Tasks

All tasks have the following top level fields:

* `id`     - (string) A url friendly identity for this node, it has to be unique within a flow. If an id is not give then the name will be used to generate the ID.
* `name`   - (string) A human friendly name to display in the web interface. If a name is not given then one will be generated from the id (either a `name` or `id` must be given).
* `class`  - There are two task classes:  
    * `task`  - A standard task does something - this is the default, and does not need to be in the config explicitly.
    * `merge` - A merge task waits for all or one of a list of events.
* `listen` - (string) The event tag that will trigger this 

Standard tasks (class `task`) have the following fields.

* `ignore-fail` - Only ever send an event tag containing the `good`postfix, even if the node failed. Can't be used in conjunction with `use-status`.
* `type`        - The specific type of task to execute.
    * `end`          - Special task that when reached positively indicates the flow ended.
    * `data`         - Accepts data from the web API, and is used to create web forms.
    * `timer`        - Waits a certain amount of time before firing its success event.
    * `exec`         - The main work horse, execute commands directly or via invoking a shell.
    * `fetch`        - Downloads a file over http(s).
    * `git-checkout` - Checkout a git repo
* `good`        - ([]int) The array of exit status codes considered a success. Default is `0` (an array of this one value)
* `use-status`  - (bool) If true then rather emit an event on task end containing the postfix `good` or `bad` use the actual exit code.
* `opts`        - (map) The variable map of options as needed by each `type`.

Merge tasks (class `merge`) have the following fields.

* `wait` - ([]string) - Array of event tags to wait for.
* `type` - The type of merge.
    * `all` - Wait for all events in the wait array.
    * `any` - Wait for any of the events in the wait array.

### Task Types

The specific task types and associated options. 

#### exec

The most common type of task node - executes a command, e.g. runs a mke command or a bash script.

Options:

* `cmd`     - Use this if you are running an executable that only depends on the binary
* `shell`   - Use this if you are running something that requires the shell, e.g. bash scripts.
* `args`    - An array of command line arguments - for simple arguments these can be included space delimited in the `cmd` or `shell` lines, if there are quote enclosed arguments then use this args array.
* `sub-dir` - The sub directory (relative to the run workspace) to execute the command in.
* `env`     - ([]string) - In the form of key=value environment variable to be set in the context of the command being executed.

#### fetch

Downloads and caches a file from the web.

Options:

* `url`           - The URL to get the file from.
* `checksum`      - The checksum to validate the file.
* `checksum-algo` - What algorithm to use to compute the checksum `sha256`, `sha1` or `md5` are supported.
* `location`      - Where to link the file once downloaded - can use `{{ws}}` substitution. Relative paths will be relative to the workspace folder for the run. If no location is given it will be linked to the root of the workspace. If the location ends in `/` (or `\` on some systems) then the file will be named as the download name, but moved to the location specified.

Development
-----------
The web assets are shipped in the binary as 'bindata' so if you change the web stuff then run `go generate ./server` to regenerate the `bindata.go`

During dev you can use the `webapp` folder directly by passing in `-dev=true`

TLS Testing
-----------
Generate a self signed cert and key and add them on the command line

```
openssl req \
    -x509 \
    -nodes \
    -newkey rsa:2048 \
    -keyout server.key \
    -out server.crt \
    -days 3650 \
    -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=*"
```

Working on a new Flow
---------------------
Working on a new flow outside of the development environment, for example if you have just downloaded `floe` or even if you have built in your go environment and want to test a flow isolated from the dev env.

### 1. From a local folder.
Launch the floe you have downloaded or built but point it at a config and root folder somewhere else. 

`floe -tags=linux,go,couch -admin=123456 -host_name=h1 -conf=/somepath/testfloe/config.yml -root=/somepath/testfloe/testfloe/ws -pub_bind=127.0.0.1:8080`

TODO 

Deploying to AWS
----------------
There is an image available that can bootstrap a floe instance `ami-006defacf6ec36202`. This image contains the Letsencrypt certbot, supervisord, git and floe.

Floe can bind its web handlers to the public and private ip's and run TLS on each independently. For instance if you are not terminating your inbound requests on a TLS enabled balancer or reverse proxy then you can bind floe to the external IP and serve TLS on that, whilst serving plain http for the floe to floe cluster.

Running floe directly on the vm means you dont benefit from the fully hermetic approach of using an ephemeral container, but floe can be used to create hermetic builds with some care and set up; the flow itself can download and install tooling into the workspace and use only these tools, of course this has an overhead, and you may want to make the tools you need to download available in S3, however many tools are well cached by amazon.

Whist floe attempts to only set env vars within the scope of its sub processes, there is nothing in particular to stop you writing scripts or programs that alter the global environment. Similarly all file activity is generally expected to be within the run workspace, but you could alter global shared storage in your flow. Given all that you may still be happy that you ave built a well controlled image that already has the tools at known versions, and you are happy that your builds are repeatable and that no action of previous builds are mutating any installed components or otherwise altering the environment, and are therefore effectively safe enough.

There is an example `start.sh` script with typical command line options. You can use this as an example of how to launch `floe` in your own vm.




