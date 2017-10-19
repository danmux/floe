import {Panel} from '../panel/panel.js';

"use strict";

// the controller for the Dashboard
export function Dash() {

    var dataReq = {
        URL: '/flows',
    }
    
    // panel is view - or part of it
    var panel = new Panel(this, null, tpl, '#main', [], dataReq);

    this.Map = function(evt) {
        console.log("dash got a call to Map", evt);

        // TODO map the event data to the panel data model
        return evt.Data;
    }

    return panel;
}

var tpl = `
<article>
    <header>
        <h1>{{=it.foo}}</h1>
        <p>No depending be convinced in unfeeling he. Excellence she unaffected and too sentiments her. Rooms he doors there ye aware in by shall. Education remainder in so cordially.</p>
    </header>
    <section>
        <h2>article section h2</h2>
        <p>For though result and talent add are parish valley. Songs in oh other avoid it hours woman style. In myself family as if be agreed. Gay collected son him knowledge delivered put. Added would end ask sight and asked saw dried house. Property expenses yourself occasion endeavor two may judgment she. Me of soon rank be most head time tore. Colonel or passage to ability.</p>
    </section>
    <section>
        <h2>article section h2</h2>
        <p>Ye on properly handsome returned throwing am no whatever. In without wishing he of picture no exposed talking minutes. Curiosity continual belonging offending so explained it exquisite. Do remember to followed yourself material mr recurred carriage. High drew west we no or at john. About or given on witty event. Or sociable up material bachelor bringing landlord confined. Busy so many in hung easy find well up.</p>
    </section>
    <footer>
        <h3>article footer h3</h3>
        <p>Ignorant branched humanity led now marianne too strongly entrance. Rose to shew bore no ye of paid rent form. Old design are dinner better nearer silent excuse. She which are maids boy sense her shade. Considered reasonable we affronting on expression in.</p>
    </footer>
</article>

<aside>
    <h3>aside</h3>
    <p>Acceptance middletons me if discretion boisterous travelling an. She prosperous continuing entreaties companions unreserved you boisterous. Middleton sportsmen sir now cordially ask additions for. You ten occasional saw everything but conviction.</p>
</aside>`