import {eventHub} from './event.js';

"use strict";

export function Controller(header, panels) {
    this.panels = panels;    // map of name -> panel
    var currentPage = {};    // name and ids for the current panel
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
    this.Activate = function(name, ids) {
        console.log('activating', name, ids);
        console.log('current-page',currentPage);
        
        // If we know we are not authenticated and not activating the login page 
        // then always redirect to the auth page
        if (!authed && name != 'login') {
            currentPage = {name: name, ids: ids};
            this.DeAuth();
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

        // make sure it has the eventHub
        panel.evtHub = eventHub;
        console.log("activated", name, panel);

        // Grab the page and ids that are becoming active.
        if (name != 'login') {
            currentPage = {name: name, ids: ids};
        }

        panel.Activate(ids);
    }

    this.DeAuth = function() {
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

    // Auth is called when the client listener detects that we have become authenticated.
    this.Auth = function() {
        // tell the header we are now authenticated
        header.Notify({
            Type: 'auth'
        });
        // remember that we are authenticated
        authed = true;

        // return to the prev page
        this.Activate(currentPage.name, currentPage.ids);
    }

    // AuthCheck - checks if the controller thinks it is authenticated
    // and calls Deauth if not.
    this.AuthCheck = function() {
        if (!authed) {
            this.DeAuth();
            return false;
        }
        return true;
    }

    // SetListener attaches a function to the Notify method which the event queue calls.
    this.SetListener = function(listener) {
        this.Notify = listener;
    }

    // NotifyPanel will send the event evt to the named panel
    this.NotifyPanel = function(name, evt) {
        var panel = this.panels[name];
        if (panel == undefined) {
            return;
        }
        panel.Notify(evt);
    }
    
    this.TrapAnchors = function (routes) {
        // set up the anchor click
        document.body.addEventListener('click', function(event) {
            var tag = event.target;
            if (event.button != 0) {
                return;
            }
            console.log(event);
    
            // find the first thing with an anchor
            while (tag.tagName != 'A') {
                if (tag.tagName == "HTML") {
                    return;
                }
                tag = tag.parentElement;
            }
            
            // It's a left click on an <a href=...> and it's a same-origin 
            // navigation: a link within the site.
            if (tag.href && (tag.origin == document.location.origin)) {
                // Now check that the the app is capable of doing a
                // within-page update. 
    
                // TODO - take .query into
                var oldPath = document.location.pathname;
                var newPath = '/app' + tag.pathname;
                // Prevent the browser from doing the navigation.
                event.preventDefault();
                // only re-route and update history if the page is new
                if (oldPath != newPath) {
                    // Let the app handle it.
                    routes(newPath);
                    history.pushState(null, '', newPath);
                }
            }
        });
    
        window.onpopstate = function(event) {
            routes(document.location.pathname);
            event.preventDefault();
        };
    }

    // subscribe this controller to the eventHub.
    eventHub.Subscribe("controller", this);
}

