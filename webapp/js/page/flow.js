import {Panel} from '../panel/panel.js';
import {Form} from '../panel/form.js';
import {RestCall} from '../panel/rest.js';
import {PrettyDate} from '../panel/util.js';

"use strict";

// the controller for the specific flow - showing all the runs
export function Flow() {
    var panel;
    var dataReq = function(){
        return {
            URL: '/flows/' + panel.IDs[0],
        };
    }

    function runClick(ev, item) {
        console.log("run summary", ev, item, item.id, item.dataset.key);
        panel.evtHub.Fire({
            Type: 'click',
            What: 'run',
            ID: item.dataset.key,
            ParentID: panel.IDs[0],
        })
    }

    var events = [
        {El: 'box.run', Ev: 'click', Fn: runClick}
    ];
    
    // panel is view - or part of it
    var panel = new Panel(this, null, tplFlow, '#main', events, dataReq);
 
    this.Map = function(evt, data) {
        console.log("flow got a call to Map", evt);

        if (evt.Type == 'rest') {
            var pl = evt.Value.Response.Payload;

            // TODO - update all these dates in the page every 30 seconds
            if (pl.Runs.Archive) {
                pl.Runs.Archive.forEach((r, i) => {
                    r.StartedAgo = PrettyDate(r.StartTime);
                    pl.Runs.Archive[i] = r;
                });
            }

            if (pl.Runs.Pending) {
                pl.Runs.Pending.forEach((r, i) => {
                    r.StartedAgo = 'waiting...';
                    pl.Runs.Pending[i] = r;
                });
            }

            if (pl.Runs.Active) {
                pl.Runs.Active.forEach((r, i) => {
                    r.StartedAgo = PrettyDate(r.StartTime);
                    pl.Runs.Active[i] = r;
                });
            }
            
            return pl;
        }

        if (evt.Type == 'ws') {
            // state changes
            if (evt.Msg.Tag == "sys.state") {
                // it was added to pending list
                if (evt.Msg.Opts.action == "add-todo") {
                    console.log("adding pending", evt.Msg);
                }
                // it was activated - so remove from pending and add to active
                if (evt.Msg.Opts.action == "activate") {
                    console.log("adding active", evt.Msg);
                    console.log(data);
                    if (data.Runs.Active == null) {
                        data.Runs.Active = [];
                    }
                    data.Runs.Active.push({
                        Ended: false,
                        Ref: evt.Msg.RunRef,
                        StartedAgo: "just now",
                        Status: "active"
                    });
                }
                return data;
            }
            // flow ended so remove it from active and add it to archive
            if (evt.Msg.Tag == "sys.end.all") {
                console.log("adding archive", evt.Msg);
                if (data.Runs.Archive == null) {
                    data.Runs.Archive = [];
                }
                var removeIndex = -1;
                for (var i = 0; i < data.Runs.Active.length; i++) {
                    if (runsEqual(evt.Msg.RunRef, data.Runs.Active[i].Ref)) {
                        removeIndex = i
                        break;
                    }
                }
                if (removeIndex >= 0) {
                    var m = data.Runs.Active[removeIndex];
                    data.Runs.Active.splice(removeIndex, 1);
                    data.Runs.Archive.unshift(m);
                }
                return data;
            }
        }
    }

    // Keep a reference to the dash panels - TODO: needed ?
    var panels = {};

    var sendData = function(data) { 
        var payload = {
            Ref: {
                ID:  panel.IDs[0],
                Ver: 1
            },
            Answers: data
        }
        RestCall(panel.evtHub, "POST", "/push/data", payload);
    }

    var trigUp = false;
    function expandHandler(elem) {
        var id = elem.getAttribute('for')
        return function(evt) {
            evt.preventDefault();
            evt.stopPropagation();

            var box = document.querySelectorAll('#trig-'+id)[0];
            var thing = document.querySelectorAll('#trig-form-container-'+id)[0]
            if (!trigUp) {
                trigUp = true;
                box.className='trigger modal';
                thing.className='trig-form expander';
                setTimeout(()=>{
                    thing.className='trig-form expander expand';
                    elem.className='expander expand'
                }, 20);
            } else {
                box.className='trigger';
                thing.className='trig-form expander';
                setTimeout(()=>{
                    thing.className='trig-form expander hidden';
                    elem.className='expander'
                }, 490);
                trigUp = false;
            }
        }
     }

    function attacheExpander() {
        var els = document.querySelectorAll('label.expander');
        var len = els.length;
        for (var i = 0; i < len; i++) {
            var elem = els[i];
            elem.addEventListener('click', expandHandler(elem));
        }
    }

    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
        if (data == undefined) {
            return
        }
        console.log(data);
        var trigs = data.Data.Config.Triggers;
        for (var t in trigs) {
            var trig = trigs[t];
            var form = trig.Opts.form;
            if (form == undefined) {
                continue;
            }
            // Give the form the trigger id so it can be uniquely directly referenced.
            form.ID = trig.ID;
            console.log(form);
            var formP = new Form('#trig-form-container-'+trig.ID, form, sendData);
            formP.Activate();
        }

        attacheExpander();
    }

    return panel;
}

function runsEqual(r1, r2) {
    return r1.FlowRef.ID == r1.FlowRef.ID &&
    r1.FlowRef.Ver == r1.FlowRef.Ver &&
    r1.Run.HostID == r1.Run.HostID &&
    r1.Run.ID == r1.Run.ID;
}

var tplFlow = `
    <div id='flow' class='flow-single'>
        <summary>
            <h2>{{=it.Data.Config.Name}}</h3>
        </summary>
        
        <div class='triggers section'>
            <heading>
                Triggers
            </heading>
            {{~it.Data.Config.Triggers :trigger:index}}
                <box id='trig-{{=trigger.ID}}' class='trigger'>
                    <h3>{{=trigger.Name}}</h4>
                    <detail>
                        {{? trigger.Type=='data'}}
                        <label for="{{=trigger.ID}}" class='expander'>Input</label>
                        <section class='trig-form expander hidden' id='trig-form-container-{{=trigger.ID}}'></section>
                        {{?}}
                    </detail>
                </box>
            {{~}}
        </div>

        <div class='active section'>
            <heading>
                Active
            </heading>
            {{~it.Data.Runs.Active :run:index}}
            <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='run' data-key='{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}'>
                <top>
                    <h4>{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}</h4>
                    <span class="label label-danger">New</span>
                </top>
                <detail>
                    {{=run.StartedAgo}}
                </detail>
            </box>
            {{~}}
        </div>

        <div class='pending section'>
            <heading>
                Pending
            </heading>
            {{~it.Data.Runs.Pending :run:index}}
            <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='run' data-key='{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}'>
                <top>
                    <h4>{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}</h4>
                    <span class="label label-danger">New</span>
                </top>
                <detail>
                    {{=run.StartedAgo}}
                </detail>
            </box>
            {{~}}
        </div>

        <div class='archive section'>
            <heading>
                Archive
            </heading>
            {{~it.Data.Runs.Archive :run:index}}
            <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='run' data-key='{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}'>
                <top>
                    <h4>{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}</h4>
                    <span class="label label-danger">New</span>
                </top>
                <detail>
                    {{=run.StartedAgo}}
                </detail>
            </box>
            {{~}}
        </div>

    </div>
`