import {Panel} from '../panel/panel.js';

"use strict";

export function Settings() {
    var panel = new Panel(
        this,
        {foo: 'with poop'}, 
        '<h1>Here is a settings template {{=it.foo}}</h1>', 
        '#main',
        {}
    );

    this.Map = function(evt) {
        console.log("settings got an Map call", evt);
        return {};
    }

    return panel;
}