import {rlite} from './vendor/rlite.js';
import {Controller} from './panel/controller.js';
import {el} from './panel/panel.js';
import {WsHub} from './ws.js';

import {Header} from './page/header.js';
import {Login} from './page/login.js';
import {Dash} from './page/dash.js';
import {Flow} from './page/flow.js';
import {FlowSingle} from './page/flow-single.js';
import {Settings} from './page/settings.js';

"use strict";

function main() {
    
    var controller = new Controller(new Header(), {
        'login'      : new Login(),
        'dash'       : new Dash(),
        'flow'       : new Flow(),       // flow triggers and history
        'flow-single': new FlowSingle(), // single flow - details - or an individual run
        'settings'   : new Settings()
    });

    controller.Base = '/app';
    
    const routes = rlite(notFound, '/app', {
        '/dash': function () { 
            controller.Activate('dash');
        },
        '/flows/:fid': function (par) { 
            controller.Activate('flow', [par.fid]);
        },
        '/flows/:fid/runs/:rid': function (par) { 
            controller.Activate('flow-single', [par.fid, par.rid]);
        },
        '/settings': function () { 
            controller.Activate('settings');
        }
    });

    var ws = new WsHub();

    // in page links also call the routes
    controller.TrapAnchors(routes);

    controller.SetListener(function(evt) {
        console.log("controller got an event", evt);

        // rest type events are fired when we get a rest response
        if (evt.Type == 'rest') {
            if (evt.Value.Status >= 500) {
                console.log("got 5xx");
                // TODO consider moving to controller - or a specific error panel?
                var wrap  = el('#err-wrap');
                wrap.className = "error show";
                el('#err-msg').innerHTML = "Call to server resulted in: " + evt.Value.Status;
                el('#err-content').innerHTML = evt.Value.Response;
                
                return
            }

            // Did we try and do a server side call and it was authenticated
            // or an explicit logout was effective, then tell the controller to UnAuthenticate
            if ((evt.Value.Status == 401) || (evt.Value.Url == '/logout' && evt.Value.Status == 200)) {
                console.log("UNAUTH");
                ws.Close();
                // DeAuth and return to the panel we were on
                controller.DeAuth();
                return;
            }

            if (evt.Value.Status == 404) {
                console.log("rest call returned 404");
                // this.Activate('problem'); // TODO - error page
                return
            }

            // did we get a successful login
            if (evt.Value.Url == '/login' && evt.Value.Status == 200) {
                console.log("LOGIN");
                ws.Close();
                ws = new WsHub();
                controller.Auth();
                return;
            }

            // map the rest event to the panel
            var panel = '';
            if (evt.Value.Url.indexOf("/runs/") >= 0) {
                panel = 'flow-single'
            } else if (evt.Value.Url.indexOf("/flows/") >= 0) {
                panel = 'flow'
            } else if (evt.Value.Url== "/flows") {
                panel = 'dash';
            }
            if (panel != "") {
                controller.NotifyPanel(panel, evt)
            }
        }

        // a specific click event was dispatched
        if (evt.Type == 'click') {
            console.log("click", evt.What, evt.ID);
            // if we know we are not authenticated then always redirect to the auth page
            if (!controller.AuthCheck()) {
                return;
            }
            if (evt.What == 'flow') {
                history.pushState(null, '', this.Base + "/flows/" + evt.ID);
                controller.Activate('flow', [evt.ID]);
            }
            if (evt.What == 'run') {
                console.log(evt.ParentID, evt.ID);
                history.pushState(null, '', this.Base + "/flows/" + evt.ParentID + "/runs/" + evt.ID);
                controller.Activate('flow-single', [evt.ParentID, evt.ID]);
            }
            if (evt.What == 'settings') {
                history.pushState(null, '', this.Base + "/settings");
                controller.Activate('settings');
            }
        }

        // web socket received message
        if (evt.Type == "ws") {
            // dash and flow need to know about state changes 
            if (
                evt.Msg.Tag == "sys.node.update" || 
                evt.Msg.Tag == "sys.state" || 
                evt.Msg.Tag =="sys.end.all" || 
                evt.Msg.Tag =="sys.node.start" ||
                evt.Msg.Tag =="sys.data.required" ||
                evt.Msg.Tag.startsWith("task") ||
                evt.Msg.Tag.startsWith("merge")
            ) {
                    controller.NotifyPanel("flow", evt);
                    controller.NotifyPanel("flow-single", evt);
            }
        }
    });
    
    // route the current path
    routes(location.pathname);
}

function notFound() {
    return '<h1>404 Not found :/</h1>';
}

main();