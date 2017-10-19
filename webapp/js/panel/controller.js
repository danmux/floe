import {eventHub} from './event.js';

"use strict";

export function Controller(header, panels) {
    this.panels = panels; // map of name -> panel
    var authRedirectTo = '';

    // always show the header
    header.evtHub = eventHub;
    header.Activate() 

    // Activate deactivates all but the requested panel and activated the named panel.
    this.Activate = function(name) {
        console.log('activating', name);

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
        panel.Activate();
    }

    this.WhichIsActive = function() {
        for (var key in this.panels) {
            if (this.panels[key].active) {
                return key;
            }
        }
        return "";
    }

    // Notify allows this controller to attach to an event hub.
    this.Notify = function(evt) {
        console.log("controller got an event", evt);

        // ----------- Users event mapping --------------

        if (evt.Type == 'rest') {
            // did we try and do a server side call and it was authenticated
            // or an explicit logout was effective
            if ((evt.Value.Status == 401) || (evt.Value.Url == '/logout' && evt.Value.Status == 200)) {
                console.log("UNAUTH");
                
                // store last page to return to after auth
                authRedirectTo = this.WhichIsActive();
                
                // notify the header we are not authenticated
                header.Notify({
                    Type: 'unauth'
                });
    
                // show the login page
                this.Activate('login');
                return;
            }
            // did we get a successful login
            if (evt.Value.Url == '/login' && evt.Value.Status == 200) {
                console.log("LOGIN");
                header.Notify({
                    Type: 'auth'
                });
                // return to the prev page
                this.Activate(authRedirectTo);
            }

            // TODO map the event to the panel
            var panel = this.panels['dash'];
            panel.Notify(evt);
        }
    }

    // subscribe this controller to the eventHub.
    eventHub.Subscribe(this);
}
