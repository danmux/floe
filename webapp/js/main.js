import {rlite} from './vendor/rlite.js';
import {Controller} from './panel/controller.js';
import {Dash} from './page/dash.js';
import {Settings} from './page/settings.js';

"use strict";

function main() {
    
    var controller = new Controller(eventHub, {
        'dash':     new Dash(),
        'settings': new Settings()
    });
    
    const routes = rlite(notFound, '/app', {
        '/dash': function () { 
            controller.Activate('dash');
        },
        '/settings': function () { 
            controller.Activate('settings');
        }
    });

    // in page links also call the routes
    trapAnchors(routes);

    // route the current path
    routes(location.pathname);
}

var dataHub = {
    'dash': {},
    'settings': {}
}

var eventHub = {
    subs: [],
}

eventHub.Subscribe = function(subscriber) {
    this.subs.push(subscriber);
}

eventHub.Fire = function(evt) {
    for (const sub of this.subs) {
        sub.Notify(evt);
    }
}

function notFound() {
    return '<h1>404 Not found :/</h1>';
}

function trapAnchors(routes) {
    // set up the anchor click
    document.body.addEventListener('click', function(event) {
        var tag = event.target;
        if (tag.tagName == 'A' && tag.href && event.button == 0) {
        // It's a left click on an <a href=...>.
            if (tag.origin == document.location.origin) {
                // It's a same-origin navigation: a link within the site.
        
                // Now check that the the app is capable of doing a
                // within-page update. 

                // TODO - take .query into
                var oldPath = document.location.pathname;
                var newPath = tag.pathname;
                // Prevent the browser from doing the navigation.
                event.preventDefault();
                // only re-route and update history if the page is new
                if (oldPath != newPath) {
                    // Let the app handle it.
                    routes(newPath);
                    history.pushState(null, '', newPath);
                }
            }
        }
    });

    window.onpopstate = function(event) {
        routes(document.location.pathname);
        event.preventDefault();
    };
}


main();