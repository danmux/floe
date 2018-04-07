import { eventHub } from './panel/event.js';

"use strict";

export function WsHub() {

    var l = document.location
    var proto = 'wss:'
    if (l.protocol == 'http:') {
        proto = 'ws:'
    }
    var wsURL = proto + "//" + l.host + "/ws"

    this.Notify = function(event) {
        // TODO - forward any WS events
    }

    this.Close = function() {
        ws.close();
    }

    // subscribe this controller to the eventHub.
    eventHub.Subscribe("ws", this);

    var ws = new WebSocket(wsURL);
    
    ws.onopen = () => {
        ws.send("Message to send"); // TODO - do we need any kind of handshake message ?
    };
    
    ws.onmessage = (evt) => { 
        eventHub.Fire({
            Type: "ws",
            Msg:JSON.parse(evt.data)
        });
    };
    
    ws.onclose = () => { 
        console.log("Connection is closed..."); 
    };
        
    window.onbeforeunload = () =>{ ws.close(); };
}

