import {Panel} from '../panel/panel.js';
import {RestCall} from '../panel/rest.js';

"use strict";

// the controller for the Dashboard
export function Header() {
    var panel = {};

    function evtLogout() {
        RestCall(panel.evtHub, "POST", "/logout");
    }

    var events = [
        {El: '#logout', Ev: 'click', Fn: evtLogout}
    ];

    panel = new Panel(this, {}, tpl, 'header', events);

    // check if we have a session cookie
    console.log(document.cookie);
    var i = document.cookie.indexOf("floe-sesh=");
    if (i >= 0) {
        console.log("got floe sesh");
        panel.store.Update("Authed", true);
    }

    this.Map = function(evt) {
        console.log('header got a call to Map', evt);

        var data = {};

        if (evt.Type == 'unauth') {
            console.log('header knows about being unauthorised');
            data.Authed = false;
        }
        if (evt.Type == 'auth') {
            console.log('header knows about being authorised');
            data.Authed = true;
        }
        // TODO map the event data to the panel data model
        return data;
    }

    return panel;
}

var tpl = `
<h3 class="title"><a href="dash">Dash</a> > Build BE</h3>
<nav>
    <ul>
        <li><a href="settings">settings</a></li>
        <li><a href="#">nav ul li a</a></li>
        {{? it.Authed }}
        <li><a id="logout" href="#">Logout</a></li>
        {{?}}
        {{? !it.Authed }}
        <li><a id="login" href="#">Login</a></li>
        {{?}}
    </ul>
</nav>
`