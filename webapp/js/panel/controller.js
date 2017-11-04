import {eventHub} from './event.js';

"use strict";

export function Controller(header, panels) {
    this.panels = panels;    // map of name -> panel
    var authRedirectTo = ''; // set in the call to deauth 
    var authed = false;      // controller maintains some state about current authentication status

    // make sure the header has access to the event hub as well
    header.evtHub = eventHub;

    // check if we have a session cookie
    console.log(document.cookie);
    var i = document.cookie.indexOf("floe-sesh=");
    if (i >= 0) {
        console.log("got floe sesh");
        authed = true; // assume token is valid - time will tell
        header.Notify({
            Type: 'auth'
        });
    }

    // always show the header
    header.Activate() 

    // check if we have a session cookie
    console.log(document.cookie);
    var i = document.cookie.indexOf("floe-sesh=");
    if (i >= 0) {
        console.log("got floe sesh");
        // assume token is valid - time will tell if it is
        header.Notify({
            Type: 'auth'
        });
        authed = true;
    }

    // Activate deactivates all but the requested panel and activated the named panel.
    this.Activate = function(name, id, par) {
        console.log('activating', name, id, par);

        // if we know we are not authenticated and not activating the login page 
        // then always redirect to the auth page
        if (!authed && name != 'login') {
            authRedirectTo = name;
            this.deauth();
            return;
        }

        // deactivate all panels except the one we want to activate.
        for (var key in this.panels) {
            if (key == name) {
                continue;
            }
            this.panels[key].Deactivate();
        }

        // activate the requested panel.
        var panel = this.panels[name];
        if (panel == undefined) {
            console.log("ERROR - missing panel", name);
            return;
        }

        panel.evtHub = eventHub;
        console.log("activated", name, panel);
        panel.Activate(id, par);
    }

    this.whichIsActive = function() {
        for (var key in this.panels) {
            if (this.panels[key].active) {
                return key;
            }
        }
        return "";
    }


    this.deauth = function() {
        // notify the header we are not authenticated
        header.Notify({
            Type: 'unauth'
        });

        // keep a record here that we are unauthenticated
        authed = false;

        // trash all data for each panel
        for (var key in this.panels) {
            this.panels[key].WipeData();
        }

        // show the login page
        this.Activate('login');
    }

    // Notify allows this controller to attach to an event hub.
    this.Notify = function(evt) {
        console.log("controller got an event", evt);

        if (evt.Type == 'rest') {
            // did we try and do a server side call and it was authenticated
            // or an explicit logout was effective
            if ((evt.Value.Status == 401) || (evt.Value.Url == '/logout' && evt.Value.Status == 200)) {
                console.log("UNAUTH");
                // deauth and return to the panel we were on
                authRedirectTo = this.whichIsActive();
                this.deauth();
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
                // tell the header we are now authenticated
                header.Notify({
                    Type: 'auth'
                });
                // remember that we are authenticated
                authed = true;
                // return to the prev page
                this.Activate(authRedirectTo);
                return;
            }

            // map the rest event to the panel
            var panel;
            if (evt.Value.Url.indexOf("/flows/") >= 0) {
                panel = this.panels['flow']
            } else{
                panel = this.panels['dash'];
            }
            panel.Notify(evt);
        }

        if (evt.Type == 'click') {
            console.log("click", evt.ID);
            // if we know we are not authenticated then always redirect to the auth page
            if (!authed) {
                authRedirectTo = this.whichIsActive();
                this.deauth();
                return;
            }
            if (evt.What == 'flow') {
                history.pushState(null, '', this.Base + "/flows/" + evt.ID);
                this.Activate('flow', evt.ID);
            }
        }
    }

    // subscribe this controller to the eventHub.
    eventHub.Subscribe(this);
}
