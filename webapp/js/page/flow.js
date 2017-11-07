import {Panel} from '../panel/panel.js';
import {Form} from '../panel/form.js';
import {RestCall} from '../panel/rest.js';

"use strict";

// the controller for the Dashboard
export function Flow() {
    var panel;
    var dataReq = function(){
        return {
            URL: '/flows/' + panel.IDs[0],
        };
    }
    
    // panel is view - or part of it
    var panel = new Panel(this, null, tplFlow, '#main', [], dataReq);

    this.Map = function(evt) {
        console.log("flow got a call to Map", evt);
        return evt.Value.Response.Payload;
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
        // label for="trig-form-container-{{=trigger.ID}}" class='expander'
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
            var form = new Form('#trig-form-container-'+trig.ID, form, sendData);
            form.Activate();
        }

        attacheExpander();
    }


    return panel;
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
            <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='active'>
                <h4>{{=run.Ref.Run.ID}}</h4>
            </box>
        {{~}}
        </div>

        <div class='pending section'>
        <heading>
            Pending
        </heading>
        {{~it.Data.Runs.Active :run:index}}
            <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='pending'>
                <h4>{{=run.Ref.Run.ID}}</h4>
            </box>
        {{~}}
        </div>

        <div class='archive section'>
            <heading>
                Archive
            </heading>
            {{~it.Data.Runs.Archive :run:index}}
                <box id='run-{{=run.Ref.Run.HostID}}-{{=run.Ref.Run.ID}}' class='archive'>
                    <h4>{{=run.Ref.Run.ID}}</h4>
                </box>
            {{~}}
        </div>

    </div>
`