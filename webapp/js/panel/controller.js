import {eventHub} from './event.js';

"use strict";

export function Controller(header, panels) {
    this.panels = panels; // map of name -> panel
    var authRedirectTo = '';
    var authed = false;

    // always show the header
    header.evtHub = eventHub;
    header.Activate() 

    // Activate deactivates all but the requested panel and activated the named panel.
    this.Activate = function(name, id, par) {
        console.log('activating', id, par);

        // if we know we are not authenticated and not activating the login page 
        // then always redirect to the auth page
        if (!authed && name != 'login') {
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
        panel.evtHub = eventHub;
        console.log("activated", panel);
        panel.Activate(id, par);
    }

    this.WhichIsActive = function() {
        for (var key in this.panels) {
            if (this.panels[key].active) {
                return key;
            }
        }
        return "";
    }

    this.deauth = function() {
        // store last page to return to after auth
        authRedirectTo = this.WhichIsActive();

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
