"use strict";

export function store(initial) {
    this.changed = true;
    this.data = initial;
    this.invalid = true;

    // Update updates the data at the given key and marks the Store as having a change.
    this.Update = function(key, val) {
        this.data[key] = val;
        this.changed = true;
    }

    // Get returns the data at the given key 
    this.Get = function() {
        if (!this.changed) {
            return null;
        }
        this.changed = false;
        return this.data;
    }
}