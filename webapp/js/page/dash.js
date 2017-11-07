import {Panel} from '../panel/panel.js';

"use strict";

// the controller for the Dashboard
export function Dash() {
    var panel = {};

    var dataReq = {
        URL: '/flows',
    }
    
    function flowSummaryClick(ev, item) {
        console.log("clicked summary", ev, item, item.id);
        panel.evtHub.Fire({
            Type: 'click',
            What: 'flow',
            ID: item.id,
        })
    }

    var events = [
        {El: 'box.flow', Ev: 'click', Fn: flowSummaryClick}
    ];

    // panel is view - or part of it
    panel = new Panel(this, null, tplDash, '#main', events, dataReq);

    this.Map = function(evt) {
        console.log("dash got a call to Map", evt);
        var flows = evt.Value.Response.Payload.Flows;

        console.log(flows);

        return {Flows: flows};
    }

    // Keep a reference to the dash panels - TODO: needed ?
    var panels = {};

    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
        // ignore if initial rendering before data fetched.
        if (data == null ) {
            return;
        }
        console.log(data.Data);
        var flows = data.Data.Flows;
        for (var f in flows) {
            var fl = flows[f];
            panels[fl.ID] = new summary(fl);
            panels[fl.ID].Activate();
        }
    }
    return panel;
}

var tplDash = `
{{~it.Data.Flows :flow:index}}
    <box id='{{=flow.ID}}' class='flow'>
    </box>
{{~}}`


function summary(flow) {
    
    // summary panel
    var panel = new Panel(this, flow, tplSummary, '#'+flow.ID, []);

    this.Map = function(evt) {
        console.log("summary got a call to Map", evt);
        return {};
    }
    return panel;
}

var tplSummary = `
    <subtitle>({{=it.Data.ID}})</subtitle>    
    <h3>{{=it.Data.Name}}</h3>
    <detail></detail>    
`
    