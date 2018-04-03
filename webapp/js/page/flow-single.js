import {Panel} from '../panel/panel.js';
import {el} from '../panel/panel.js';
import {AttacheExpander} from '../panel/expander.js';
import {PrettyDate} from '../panel/util.js';

"use strict";

// the controller for the Dashboard
export function FlowSingle() {
    var panel;
    var dataReq = function(){
        return {
            URL: '/flows/' + panel.IDs[0] + '/runs/' + panel.IDs[1],
        };
    }
    
    var events = [];

    // panel is view - or part of it
    var panel = new Panel(this, null, graphFlow, '#main', events, dataReq);

    this.Map = function(evt) {
        console.log("flow got a call to Map", evt);
        if (evt.Type == 'rest') {
          var pl = evt.Value.Response.Payload;
          pl.Parent = '/flows/' + panel.IDs[0];
          console.log(pl);

          pl.Graph.forEach((r, i) => {
              r.forEach((nr, ni) => {
                nr.StartedAgo = PrettyDate(nr.Started);
                nr.Took = "";
                if ( nr.Stopped != "0001-01-01T00:00:00Z" ) {
                    var started = new Date(nr.Started)
                    var stopped = new Date(nr.Stopped);
                    nr.Took = "("+toHHMMSS((stopped - started)/1000)+")";
                }
                pl.Graph[i][ni] = nr;
              });
          });

          console.log(pl);

          return pl;
        }
    }

    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
      AttacheExpander(el('triggers'));
    }

    return panel;
}

function toHHMMSS(sec_num) {
    sec_num = Math.floor(sec_num)
    var hours   = Math.floor(sec_num / 3600);
    var minutes = Math.floor((sec_num - (hours * 3600)) / 60);
    var seconds = sec_num - (hours * 3600) - (minutes * 60);

    
    if (minutes < 10) {minutes = "0"+minutes;}
    if (seconds < 10) {seconds = "0"+seconds;}
    if (hours > 0) {
        return hours+':'+minutes+':'+seconds;
    }
    return minutes+':'+seconds;
}

var graphFlow = `
    <div id='flow' class='flow-single'>
        <div class="crumb">
          <a href='{{=it.Data.Parent}}'>← back to {{=it.Data.FlowName}}</a>
        </div>
        <summary>
            <h3>{{=it.Data.Name}}</a></h3>
        </summary>
        <triggers>
          {{~it.Data.Triggers :trigger:index}}

          <box id='trig-{{=trigger.ID}}' class='trigger{{? !trigger.Enabled}} disabled{{?}}'>
              {{? trigger.Type=='data'}}
              <div for="{{=trigger.ID}}" class="trig-title expander-ctrl">
                  <h4>{{=trigger.Name}}</h4><i class='icon-angle-circled-right'></i>
              </div>
              {{??}}
              <div class="trig-title">
                  <h4>{{=trigger.Name}}</h4>
              </div>
              {{?}}
              {{? trigger.Type=='data'}}
              <detail id='expander-{{=trigger.ID}}' class='expander'>
                  {{~trigger.Fields :field:index}}
                    <div id="field-{{=field.ID}}", class='kvrow'>
                      <div class='prompt'>{{=field.Prompt}}:</div> 
                      <div class='value'>{{=field.Value}}</div>
                    </div>
                  {{~}}
              </detail>
              {{?}}
          </box>

          {{~}}
        </triggers>
        <divider></divider>
        <tasks>
        {{~it.Data.Graph :level:index}}
          <div id='level-{{=index}}' class='level'>
          {{~level :node:indx}}
            <box id='node-{{=node.ID}}' class='task {{=node.Result}} {{=node.Status}}'>
              <h4>{{=node.Name}}</h4>
              <detail>
                <p class='ago'>{{=node.StartedAgo}}</p><p class='took'>{{=node.Took}}</p>
              <detail>
            </box>
          {{~}}
          </div>
        {{~}}
        </tasks>
    </div>
`