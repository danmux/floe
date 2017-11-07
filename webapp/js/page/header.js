import {Panel} from '../panel/panel.js';
import {RestCall} from '../panel/rest.js';

"use strict";

// the controller for the Dashboard
export function Header() {
    var panel = {};

    function evtLogout() {
        RestCall(panel.evtHub, "POST", "/logout");
    }

    function evtSettings() {
        panel.evtHub.Fire({
            Type: 'click',
            What: 'settings',
        })
    }

    var events = [
        {El: '#settings', Ev: 'click', Fn: evtSettings},
        {El: '#logout', Ev: 'click', Fn: evtLogout}    
    ];

    panel = new Panel(this, {}, tpl, 'header', events);

    this.Map = function(evt) {
        var data = {};
        if (evt.Type == 'unauth') {
            data.Authed = false;
        }
        if (evt.Type == 'auth') {
            data.Authed = true;
        }
        // TODO map the event data to the panel data model
        return data;
    }

    return panel;
}

var tpl = `
<h3 class='title'><a href='/dash'>Floe</a></h3>
<nav>
    <ul>
        <li><a id='settings'><i class='icon-cog'></i></a></li>
        {{? it.Data.Authed }}
        <li><a id='logout'><i class='icon-off'></i></a></li>
        {{?}}
    </ul>
</nav>
`