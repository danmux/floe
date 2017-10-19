"use strict";

export var eventHub = {
    subs: [],
}

eventHub.Subscribe = function(subscriber) {
    this.subs.push(subscriber);
}

eventHub.Fire = function(evt) {
    console.log("EVENT:", evt.Type);
    for (const sub of this.subs) {
        sub.Notify(evt);
    }
}