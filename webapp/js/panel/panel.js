import {doT} from '../vendor/dot.js';
import {store} from './store.js';

"use strict";

function el(sel) {
    return document.querySelectorAll(sel);
}

// Panel object
export function Panel(parent, data, template, attach) {
    
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
        this.Render(true); // render it in even if the data is unchanged
    }

    // Deactivate marks this panel as not active. Any subsequent calls to render will return without changing the dom.
    this.Deactivate = function() {
        this.active = false;
    }

    // Notify is called by the controller of this panel 
    this.Notify = function(evt) {
        console.log("panel got an event", evt);

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
        var resultText = this.tempFn(this.store.data);
        el(attach)[0].innerHTML = resultText; // TODO - trap missing el?
        
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