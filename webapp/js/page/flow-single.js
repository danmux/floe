import {Panel} from '../panel/panel.js';

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

    var panels = {};

    return panel;
}

var graphFlow = `
    <div id='flow' class='flow-single'>
        <summary>
            <h3><a href='{{=it.Data.Parent}}'> {{=it.Data.Config.Name}}</a></h3>
        </summary>
        <triggers>
        {{~it.Data.Config.Triggers :trigger:index}}
            <box id='trig-{{=trigger.ID}}' class='trigger'>
                <h4>{{=trigger.Name}}</h4>
            </box>
        {{~}}
        </triggers>
        <divider></divider>
        <tasks>
        {{~it.Data.Graph :level:index}}
          <div id='level-{{=index}}' class='level section'>
          {{~level :trigger:indx}}
            <box id='trig-{{=trigger}}' class='task good'>
              <h4>{{=trigger}}</h4>
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