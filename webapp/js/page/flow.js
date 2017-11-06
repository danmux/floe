import {Panel} from '../panel/panel.js';
import {Form} from '../panel/form.js';

"use strict";

// the controller for the Dashboard
export function Flow() {
    var panel;
    var dataReq = function(){
        return {
            URL: '/flows/' + panel.IDs[0],
        };
    }
    
    var events = [];

    // panel is view - or part of it
    var panel = new Panel(this, null, tplFlow, '#main', events, dataReq);

    this.Map = function(evt) {
        console.log("flow got a call to Map", evt);
        return evt.Value.Response.Payload.Config;
    }

    // Keep a reference to the dash panels - TODO: needed ?
    var panels = {};
    
    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
        if (data == undefined) {
            return
        }
        console.log(data);
        var trigs = data.Data.Triggers;
        for (var t in trigs) {
            var trig = trigs[t];
            var form = trig.Opts.form;
            if (form == undefined) {
                continue;
            }
            // Give the form the trigger id so it can be uniquely directly referenced.
            form.ID = trig.ID;
            console.log(form);
            var form = new Form('#trig-form-container-'+trig.ID, form, (e)=>{console.log(e)});
            form.Activate();
        }
    }


    return panel;
}

var tplFlow = `
    <div id='flow' class='flow-single'>
        <summary>
            <h3>{{=it.Data.Name}}</h3>
        </summary>
        
        <triggers>
        {{~it.Data.Triggers :trigger:index}}
            <box id='trig-{{=trigger.ID}}' class='trigger'>
                <h4>{{=trigger.Name}}</h4>
                <div class='trig-form' id='trig-form-container-{{=trigger.ID}}'></div>
            </box>
        {{~}}
        </triggers>

        <history>
        
        </tasks>

    </div>
`
