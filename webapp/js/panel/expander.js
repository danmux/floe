import { store } from "./store.js";

"use strict";

var expandState = new store({});

function expandHandler(pageID, elem) {
    var opened = false;
    var id = elem.getAttribute('for')
    var thing = document.querySelectorAll('#expander-'+id)[0];
    var origThingClass = thing.className.replace(" expand", "");
    var ctrlI = elem.querySelectorAll('i')[0];
    var origCtrlIClass = ctrlI.className.replace(" open", "");
    
    return function(evt) {
        evt.preventDefault();
        evt.stopPropagation();

        var pageState = States(pageID);
        opened = pageState[id];

        if (!opened) {
            opened = true;
            setTimeout(()=>{
                thing.className = thing.className + ' expand';
                ctrlI.className = origCtrlIClass + ' open';
            }, 20);
        } else {
            setTimeout(()=>{
                thing.className = origThingClass;
                ctrlI.className = origCtrlIClass;
            }, 20);
            opened = false;
        }
        pageState[id] = opened;
        expandState.Update(pageID, pageState);
    }
 }

export function States(pageID) {
    var states = expandState.Get(true);
    var pageState = states[pageID];
    if (pageState==undefined) {
        pageState = {}
    }
    return pageState;
}

export function AttacheExpander(id, root) {
    var els = root.querySelectorAll('.expander-ctrl');
    var len = els.length;
    for (var i = 0; i < len; i++) {
        var elem = els[i];
        elem.addEventListener('click', expandHandler(id, elem));
    }
}