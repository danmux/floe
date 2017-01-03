floe
====

floe - code over convention workfloe engine - think gocd but actually in go - oh and no xml


floe 
----
A description of a sequence of nodes joined by events

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






