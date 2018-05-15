import {Panel} from '../panel/panel.js';
import {RestCall} from '../panel/rest.js';

"use strict";

// the controller for the Dashboard
export function Login() {
    var panel = {};

    function login() {
        console.log("submitted login");
        var payload = {
            User:     "admin",
            Password: "password"
        }

        RestCall(panel.evtHub, "POST", "/login", payload);
    }

    var events = [
        {El: 'button[name="Submit"]', Ev: 'click', Fn: login}
    ];

    panel = new Panel(this, {}, tpl, '#main', events);

    this.Map = function(evt) {
        console.log("auth got a call to Map", evt);

        // TODO map the event data to the panel data model
        return evt.Data;
    }

    return panel;
}

var tpl = `
    <div class='login'>
        <form action=' method='post' name='login-form' class='form-signin'>       
            <h3 class='form-signin-heading'>Please log in</h3>
            <hr>
            
            <input type='text' name='Username' placeholder='Username' required='' autofocus='' />
            <input type='password' name='Password' placeholder='Password' required=''/>     		  
            
            <button class='btn' name='Submit' value='Login' type='Submit'>Login</button>  			
        </form>			
    </div>
`