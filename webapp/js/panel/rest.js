"use strict";

export function RestCall(evtHub, method, url, obj) {
    var apiUrl = '/build/api'+url;

    var body = null;
    
    var xhr = new XMLHttpRequest();
    xhr.open(method, apiUrl, true);

    xhr.setRequestHeader('Accept', 'application/json')

    if ((method == 'PUT' || method == 'POST') && obj != undefined) {
        xhr.setRequestHeader('Content-Type', 'application/json')
        body = JSON.stringify(obj);
    }

    xhr.ontimeout = function () {
        console.error("The request for " + apiUrl + " timed out.");
        evtHub.Fire({
            Type: 'top-error',
            Value: "Request timed out"
        });
    };

    xhr.onload = function() {
        if (xhr.readyState === 4) {
            var resp = {
                Url: url,
                Status: xhr.status,
                Response: {}
            };
            var ct = xhr.getResponseHeader("Content-Type");
            if(ct && ct.includes("application/json")) {
                resp.Response = JSON.parse(xhr.response);
            }
            evtHub.Fire({
                Type: 'rest',
                Value: resp,
            });
        }
    };

    xhr.timeout = 5000; // 5 second timeout
    xhr.send(body);
}