floe
====

floe - code over convention workfloe engine - think gocd but actually in go - oh and no xml


start three terminals:

1. floe -tags=linux,go,couch -admin=123456 -host=h1 -bind=127.0.0.1:8080

2. floe -tags=linux,go,couch -admin=123456 -host=h2 -bind=127.0.0.1:8090

3. one for dev



floe 
----
Flow - A description of a set of Nodes linked by events. 

Node - A config item that can respond to and issue events. Certain Nodes can Execute actions
The types of Node are 

Triggers - These are http or polling nodes that respond to http requests or changes in a polled entity.
Tasks - Nodes that on responding to an event execute some action, and then emit an event.
Pubs - Publishers are nodes that emit a published event to a third party. 
Merges - a Node that waits for a certain number of events before issuing its event.

Run - A specific invocation of a flow.


Hub  - is the central routing object, it instantiates tasks based on its config from any events it observes.
Flow - a specific configuration of tasks

Pending queue


run
---
a specific execution of a floe



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






