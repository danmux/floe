import {Panel} from '../panel/panel.js';
import {el} from '../panel/panel.js';
import {AttacheExpander} from '../panel/expander.js';

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
        // return evt.Value.Response.Payload.Config;
        evt.Value.Response.Payload.Parent = '/flows/' + panel.IDs[0];
        console.log(evt.Value.Response.Payload);
        return evt.Value.Response.Payload;
    }

    // AfterRender is called when the dash hs rendered containers.
    // we go and add the child summary panels
    this.AfterRender = function(data) {
      AttacheExpander(el('triggers'));
    }

    return panel;
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
          <div id='level-{{=index}}' class='level section'>
          {{~level :node:indx}}
            <box id='node-{{=node.ID}}' class='task good'>
              <h4>{{=node.Name}}</h4>
            </box>
          {{~}}
          </div>
        {{~}}
        </tasks>

    </div>
`


var tplFlow = `
    <div id='flow' class='flow-single'>
        <summary>
            <h3>{{=it.Data.Name}}</h3>
        </summary>
        
        <triggers>
        {{~it.Data.Triggers :trigger:index}}
            <box id='trig-{{=trigger.ID}}' class='trigger'>
                <h4>{{=trigger.Name}}</h4>
            </box>
        {{~}}
        </triggers>

        <tasks>
        {{~it.Data.Tasks :task:index}}
            <box id='task-{{=task.ID}}' class='task'>
                <h4>{{=task.Name}}</h4>
            </box>
        {{~}}
        </tasks>

    </div>
`

/*
{
  "Message": "OK",
  "Payload": {
    "Config": {
      "ID": "build-project",
      "Ver": 1,
      "Name": "build project",
      "ReuseSpace": true,
      "HostTags": [
        "linux",
        "go",
        "couch"
      ],
      "ResourceTags": [
        "couchbase",
        "nic"
      ],
      "Triggers": [
        {
          "Ref": {
            "Class": "trigger",
            "ID": "push"
          },
          "ID": "push",
          "Name": "push",
          "Listen": "",
          "Wait": null,
          "Type": "git-push",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {
            "url": "blah.blah"
          }
        },
        {
          "Ref": {
            "Class": "trigger",
            "ID": "start"
          },
          "ID": "start",
          "Name": "start",
          "Listen": "",
          "Wait": null,
          "Type": "data",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {
            "form": "-"
          }
        }
      ],
      "Tasks": [
        {
          "Ref": {
            "Class": "task",
            "ID": "checkout"
          },
          "ID": "checkout",
          "Name": "checkout",
          "Listen": "merge.subs.good",
          "Wait": null,
          "Type": "git-merge",
          "Good": [
            0
          ],
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {}
        },
        {
          "Ref": {
            "Class": "task",
            "ID": "build"
          },
          "ID": "build",
          "Name": "build",
          "Listen": "task.checkout.good",
          "Wait": null,
          "Type": "exec",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {
            "cmd": "make build"
          }
        },
        {
          "Ref": {
            "Class": "task",
            "ID": "test"
          },
          "ID": "test",
          "Name": "test",
          "Listen": "task.build.good",
          "Wait": null,
          "Type": "exec",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {
            "cmd": "make test"
          }
        },
        {
          "Ref": {
            "Class": "task",
            "ID": "sign-off"
          },
          "ID": "sign-off",
          "Name": "Sign Off",
          "Listen": "task.build.good",
          "Wait": null,
          "Type": "data",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {
            "form": "-"
          }
        }
      ],
      "Pubs": null,
      "Merges": [
        {
          "Ref": {
            "Class": "merge",
            "ID": "subs"
          },
          "ID": "subs",
          "Name": "subs",
          "Listen": "",
          "Wait": [
            "sub.push.good",
            "sub.start.good"
          ],
          "Type": "any",
          "Good": null,
          "IgnoreFail": false,
          "UseStatus": false,
          "Opts": {}
        }
      ]
    }
  }
}
*/