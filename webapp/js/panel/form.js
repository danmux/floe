import {Panel} from './panel.js';
import {el} from './panel.js';
import {doT} from '../vendor/dot.js';

"use strict";

export function Form(sel, obj, onSubmit) {

    // grab the form values to send to the callback
    var submitClick = function() {
        var data = {
            ID: obj.ID,
            Values: {},
        };
        for (var f in obj.fields) {
            var field = obj.fields[f];
            data.Values[field.id] = el('input[name="field-'+field.id+'"]').value;
        }
        // callback with the form data
        onSubmit(data);
    } 

    var events = [
        {El: '#submit-'+obj.ID, Ev: 'click', Fn: submitClick}
    ];

    var panel = new Panel(this, obj, tplForm, sel, events);

    // Map not used as no events expected for forms
    this.Map = function(evt) {
        console.log("Form got an event", evt);
        return {};
    }

    return panel;
}

var tplForm = `
<form id="form-{{=it.Data.ID}}">
    <fields>
    {{~it.Data.fields :field:index}}
        <prompt>{{=field.prompt}}:</prompt>
        <input type="{{=field.type}}" name="field-{{=field.id}}" value="">
    {{~}}
    </fields>

    <button class="btn" id="submit-{{=it.Data.ID}}" type="submit">Submit</button>

</form>
`