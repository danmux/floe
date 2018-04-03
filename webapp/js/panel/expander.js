"use strict";

export var eventHub = {
    subs: {},
}


function expandHandler(elem) {
    var opened = false;
    var id = elem.getAttribute('for')
    var thing = document.querySelectorAll('#expander-'+id)[0];
    var origThingClass = thing.className;
    var ctrlI = elem.querySelectorAll('i')[0];
    var origCtrlIClass = ctrlI.className;
    
    return function(evt) {
        evt.preventDefault();
        evt.stopPropagation();

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
    }
 }

export function AttacheExpander(root) {
    var els = root.querySelectorAll('.expander-ctrl');
    var len = els.length;
    for (var i = 0; i < len; i++) {
        var elem = els[i];
        elem.addEventListener('click', expandHandler(elem));
    }
}