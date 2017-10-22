import {Panel} from '../panel/panel.js';

"use strict";

// the controller for the Dashboard
export function Flow() {
    var panel;
    var dataReq = function(){
        return {
            URL: '/flows/' + panel.ID,
        };
    }
    
    var events = [];

    // panel is view - or part of it
    var panel = new Panel(this, null, tplFlow, '#main', events, dataReq);

    this.Map = function(evt) {
        console.log("flow got a call to Map", evt);
        return evt.Value.Response.Payload;
    }

    var panels = {};

    return panel;
}

var tplFlow = `
    <div id='flow' class='flow-single'>
        {{=it.Data.Config.Name}}
    </div>
`
