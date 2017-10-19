import {doT} from '../vendor/dot.js';
import {RestCall} from './rest.js';
import {store} from './store.js';

"use strict";

function el(sel) {
    return document.querySelectorAll(sel);
}

// Panel object parent is provided so we can call back and map the event to the store for this panel.
// data is some initial state for the store, it has to be null if you want to trigger the restReq to
// get some initial data from the server.
export function Panel(parent, data, template, attach, events, restReq) {
    // N.B. this.evtHub is set by the controller

    this.store = new store(data);  // the data store to render this panel.
    this.template = template;      // the template to use to render the html from the store data.
    this.attach = attach;          // the CSS selector to attach the resultant html to.

    this.active = false;           // active is true if we think this panel is in the dom.

    // Compile template function
    this.tempFn = doT.template(template);

    // Activate is called to mark this panel active and render it into the dom.
    this.Activate = function() {
        if (this.active) {
            return;
        }
        console.log("activating", this);
        this.active = true;

        // if this store is empty get the data from the server.
        if (this.store.IsEmpty()) {
            this.GetData();
        }

        // render it in even if the data is unchanged, hence true param.
        this.Render(true); 
    }

    this.GetData = function() {
        if (restReq == undefined) {
            return;
        }
        if (restReq.Method == undefined ){
            restReq.Method = 'GET';
        }
        RestCall(this.evtHub, restReq.Method, restReq.URL, restReq.ReqBodyObj); 
    }

    // Deactivate marks this panel as not active. Any subsequent calls to render will return without changing the dom.
    this.Deactivate = function() {
        this.active = false;
    }

    // Notify is called by the controller of this panel 
    this.Notify = function(evt) {
        var data = parent.Map(evt);

        for (var key in data) {
            console.log("updating", key, data[key]);
            this.store.Update(key, data[key]);
            console.log(this);
            this.Render(false); // false asks for a render only if the data changed.
        }
    }

    // Render will re-render the template into the 'attach' selector, only if the data has changed.
    this.Render = function(force) {
        // only render active panels
        if (!this.active) {
            return;
        }
        
        // get the stored data - if it is unchanged it will be null
        data = this.store.Get(force);
        if (data == null && !force) {
            return;
        }
        
        // get the template and attach it to its dom element
        var resultText = "";
        if (data != null) {
            resultText = this.tempFn(data);
        }
        console.log(el(attach));
        el(attach)[0].innerHTML = resultText; // TODO - trap missing el?

        // attach all the events
        for (var i in events) {
            var event = events[i];
            console.log("ev:", event);
            el(event.El).forEach(elem => {
                console.log("adding event", event.El, event.Ev)
                elem.addEventListener(event.Ev, event.Fn);
            });
        }
    }
}

export function Store(initial) {
    this.changed = true;
    this.data = initial;
    this.invalid = true;

    // Update updates the data at the given key and marks the Store as having a change.
    this.Update = function(key, val) {
        this.data[key] = val;
        this.changed = true;
    }

    // Get returns the data at the given key 
    this.Get = function(force) {
        if (!this.changed && !force) {
            return null;
        }
        this.changed = false;
        return this.data;
    }
}