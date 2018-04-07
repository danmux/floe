import {Panel} from '../panel/panel.js';
import {el} from '../panel/panel.js';
import {Form} from '../panel/form.js';
import {RestCall} from '../panel/rest.js';
import {PrettyDate} from '../panel/util.js';
import {ToHHMMSS} from '../panel/util.js';
import {AttacheExpander} from '../panel/expander.js';

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
        console.log("flow got a call to Map", evt, data);

        if (evt.Type == 'rest') {
            var pl = evt.Value.Response.Payload;

            // TODO - update all these dates in the page every 30 seconds
            if (pl.Runs.Archive) {
                pl.Runs.Archive.forEach((r, i) => {
                    pl.Runs.Archive[i] = EmbellishSummary(r);
                });
            }

            if (pl.Runs.Pending) {
                pl.Runs.Pending.forEach((r, i) => {
                    pl.Runs.Pending[i] = EmbellishSummary(r);
                });
            }

            if (pl.Runs.Active) {
                pl.Runs.Active.forEach((r, i) => {
                    pl.Runs.Active[i] = EmbellishSummary(r);
                });
            }
            return pl;
        }

        if (evt.Type == 'ws') {
            if (data == null) { // no need to update data that has not been initialised yet
                return;
            }
            // state changes
            if (evt.Msg.Tag == "sys.state") {
                // TODO it was added to pending list
                if (evt.Msg.Opts.action == "add-pend") {
                    console.log("adding pending", evt.Msg);
                }
                // it was activated - so add to active and TODO remove from pending
                if (evt.Msg.Opts.action == "activate") {
                    console.log("adding active", evt.Msg);
                    console.log(data);
                    if (data.Runs.Active == null) {
                        data.Runs.Active = [];
                    }
                    var d = new Date();
                    data.Runs.Active.push({
                        Ended: false,
                        StartTime: d.toISOString(),
                        EndTime: "0001-01-01T00:00:00Z",
                        Ref: evt.Msg.RunRef,
                        StartedAgo: "just now",
                        Status: "running",
                        Stat: "Active",
                        Took: "(00:00)"
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
                data.Runs.Active.forEach((r, i) => {
                    if (runsEqual(evt.Msg.RunRef, r.Ref)) {
                        removeIndex = i
                        return;
                    }
                });
                if (removeIndex >= 0) {
                    var m = data.Runs.Active[removeIndex];
                    m.Ended = true;
                    var d = new Date();
                    m.EndTime = d.toISOString();
                    if (evt.Msg.Good) {
                        m.Status = "good";
                    } else {
                        m.Status = "bad";
                    }
                    m = EmbellishSummary(m);
                    data.Runs.Active.splice(removeIndex, 1);
                    data.Runs.Archive.unshift(m);
                }
                return data;
            }
            // update some stuff on the active run this event relates to.
            data.Runs.Active.forEach((r, i) => {
                if (runsEqual(evt.Msg.RunRef, r.Ref)) {
                    data.Runs.Active[i] = EmbellishSummary(r);
                    return;
                }
            });
            return data;
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
            Form: data
        }
        RestCall(panel.evtHub, "POST", "/push/data", payload);
    }

    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
        if (data == undefined) {
            return
        }
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
            var formP = new Form('#expander-'+trig.ID, form, sendData);
            formP.Activate();
        }

        AttacheExpander(el('.triggers'));
    }

    return panel;
}

export function EmbellishSummary(r) {
    switch(r.Status) {
        case "running":
            r.Stat = "Active"
            break;
        case "good":
            r.Stat = "Success"
            break;
        case "bad":
            r.Stat = "Failed"
            break;
        default:
            r.Stat = "New"
    };

    r.Took = "";
    var toTime = new Date();
    if (r.EndTime != "0001-01-01T00:00:00Z") {
        toTime = new Date(r.EndTime);
    }
    r.StartedAgo = 'waiting...';
    if (r.StartTime != "0001-01-01T00:00:00Z") {
        r.StartedAgo = PrettyDate(r.StartTime);
        var started = new Date(r.StartTime);
        r.Took = "("+ToHHMMSS((toTime - started)/1000)+")";
    }
    return r;
};

function runsEqual(r1, r2) {
    return r1.FlowRef.ID == r1.FlowRef.ID &&
    r1.FlowRef.Ver == r1.FlowRef.Ver &&
    r1.Run.HostID == r1.Run.HostID &&
    r1.Run.ID == r1.Run.ID;
}

var tplFlow = `
    <div id='flow' class='flow'>
        <div class="crumb">
          <a href='/dash'>← back to Dashboard</a>
        </div>
        <summary>
            <h2>{{=it.Data.Config.Name}}</h3>
        </summary>
        
        <div class='triggers section'>
            <heading>
                Triggers
            </heading>
            {{~it.Data.Config.Triggers :trigger:index}}
                <box id='trig-{{=trigger.ID}}' class='trigger'>
                    {{? trigger.Type=='data'}}
                    <div for="{{=trigger.ID}}" class="data-title expander-ctrl">
                        <h4>{{=trigger.Name}}</h4><i class='icon-angle-circled-right'></i>
                    </div>
                    {{??}}
                    <div class="data-title">
                        <h4>{{=trigger.Name}}</h4>
                    </div>
                    {{?}}
                    {{? trigger.Type=='data'}}
                    <detail id='expander-{{=trigger.ID}}' class='expander'>
                    </detail>
                    {{?}}
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
                    <span class="label {{=run.Status}}">{{=run.Stat}}</span>
                </top>
                <detail>
                    <p class='ago'>{{=run.StartedAgo}}</p><p class='took'>{{=run.Took}}</p>
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
                    <span class="label {{=run.Status}}">{{=run.Stat}}</span>
                </top>
                <detail>
                    <p class='ago'>{{=run.StartedAgo}}</p><p class='took'>{{=run.Took}}</p>
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
                    <span class="label {{=run.Status}}">{{=run.Stat}}</span>
                </top>
                <detail>
                    <p class='ago'>{{=run.StartedAgo}}</p><p class='took'>{{=run.Took}}</p>
                </detail>
            </box>
            {{~}}
        </div>

    </div>
`