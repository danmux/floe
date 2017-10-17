
"use strict";

export function Controller(eventHub, panels) {
    this.panels = panels; // map of name -> panel

    // Activate deactivates all but the requested panel and activated the named panel.
    this.Activate = function(name) {
        console.log('activating', name);

        // deactivate all panels except the one we want to activate.
        for (var key in this.panels) {
            if (key == name) {
                continue;
            }
            this.panels[key].Deactivate();
        }

        // activate the requested panel.
        this.panels[name].Activate();
    }

    // Notify allows this controller to attach to an event hub.
    this.Notify = function(evt) {
        console.log("controller got an event", evt);

        // TODO map the event to the panel
        var panel = this.panels['dash'];
        panel.Notify(evt);
    }

    // subscribe this controller to the eventHub.
    eventHub.Subscribe(this);
}
