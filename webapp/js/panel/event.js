"use strict";

export var eventHub = {
    subs: {},
}

eventHub.Subscribe = function(key, subscriber) {
    this.subs[key] = subscriber;
}

eventHub.Fire = function(evt) {
    console.log("EVENT:", evt.Type);
    for (const k in this.subs) {
        var sub = this.subs[k];
        sub.Notify(evt);
    }
}