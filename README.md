floe
====

floe - code over convention workfloe engine - think gocd but actually in go - oh and no xml


start three terminals:

1. floe -tags=linux,go,couch -admin=123456 -host=h1 -bind=127.0.0.1:8080

2. floe -tags=linux,go,couch -admin=123456 -host=h2 -bind=127.0.0.1:8090

3. one for dev

web 
---

http://localhost:8080/app/dash


floe 
----
Host - any compute instance running a Floe service.
Flow - A description of a set of Nodes linked by events. A specific instance of an flow is a Run

Node - A config item that can respond to and issue events. A Node is part of a flow. Certain Nodes can Execute actions. Merge 
The types of Node are. 

* Triggers - These are http or polling nodes that respond to http requests or changes in a polled entity.
* Tasks - Nodes that on responding to an event execute some action, and then emit an event.
* Pubs - Needed ? Publishers are nodes that emit a published event to a third party. 
* Merges - a Node that waits for a certain number of events before issuing its event.

Run - A specific invocation of a flow, can be in one of three states Pending, Active, Archive.
RunRef - An 'adopted' RunRef is a globally unique compound reference that resolves to a specific Run.
Hub  - is the central routing object. It instantiates Runs and executes actions on Nodes in the Run based on its config from any events it observes on its queue.
Queue - The hub event queue is the central chanel for all changes.
RunStore - the hub references a run store that can persist the following lists - representing the three states fo a run..
* Pending list - Runs waiting to be executed, a Run on this list is called a Todo.
* Active List - Runs that are executing, and will be matched to events with matching adopted RunRefs. 
* Archive List - Runs that are have finished executing.


Life cycle of a flow
--------------------
When a trigger event arrives on the queue that matches a flow, the event reference will be considered 'un-adopted' this means it has not got a full run reference. A run is created with a globally unique compound reference (now adopted) - this reference (and some other meta data) is added to the pending list of the host that adopted it as a 'Todo' - this may not be the host that executes the run.

A background process tries to assign Todo's to any host where the HostTags match, and where there are no Runs already matching the ResourceTags asked for - this allows certain nodes to be assigned to certain Runs, and to serialise Runs that need exclusive access to any third party, or other shared resources.

Once a Todo has been dispatched for execution it is moved out of the adopting Pending list and into the Active List on the executing host.

When one of the end conditions for a Run is met the Run is moved out of the Active list and into the Archive list on the host that executed the Run.



list of runs

runid, floeid


floes:
   active.json - id, curver

   id:
     ver:
        desc.yaml

        running:
           id:
              run.json run id, id, ver,
              node-1.json - append only status
        done:
           id.json
              




State changes with:

External Event ->  

Executor Completion -> 

Executor Output -> 


Engine
------
/floes/active.json
/floes/build-proj/
/floes/build-proj/1
/floes/build-proj/1/desc.json
/floes/build-proj/1/running/3/run.json
/floes/build-proj/1/running/3/node-1.json
/floes/build-proj/1/running/3/node-2.json
/floes/build-proj/1/done/5/run.json
/floes/build-proj/1/done/5/node-1.json
/floes/build-proj/1/done/5/node-2.json






