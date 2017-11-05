import {Panel} from '../panel/panel.js';

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

    var panels = {};

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
            </box>
        {{~}}
        </triggers>

        <history>
        
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