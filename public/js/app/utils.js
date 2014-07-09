"use strict";

define([
  'underscore'
], function(_) {
    var Utils = {

        getObj: function(url, cb_success, cb_fail) {
            Utils.sendData('GET', '', url, cb_success, cb_fail);
        },

        putObj: function(payload, url, cb_success, cb_fail) {
            var dat = JSON.stringify(payload);
            Utils.sendData('PUT', dat, url, cb_success, cb_fail);
        },

        sendObj: function(payload, url, cb_success, cb_fail) {
            var dat = JSON.stringify(payload);
            Utils.sendData('POST', dat,url, cb_success, cb_fail);
        },

        // TODO add token - so it works in an app
        sendData: function(type, dat, url, cb_success, cb_fail) {
            $.ajax({
                type: type,
                url: url,
                data: dat,
                contentType: 'text/json',
                dataType: 'json',
                timeout: 30000,

                success: function(data){
                    console.log(data);
                    cb_success(data)
                },
                error: function(xhr, type){
                    console.log(type);
                    console.log(xhr);
                    cb_fail(xhr)
                }
            });
        },

        validateEmail: function (email) { 
            var re = /^(([^<>()[\]\\.,;:\s@\"]+(\.[^<>()[\]\\.,;:\s@\"]+)*)|(\".+\"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
            return re.test(email);
        },

        prettyDateFromYMD: function(what) {
            var year = what.substring(0,4);
            var month = parseInt(what.substring(5,7)) -1;
            var day = what.substring(8,10);
            var d = new Date(year, month, day);

            var lang;
            if (navigator.userLanguage) // Explorer
                lang = navigator.userLanguage;
            else if (navigator.language) // FF
                lang = navigator.language;
            else
                lang = "en-GB";
            
            var options = {weekday: "long", year: "numeric", month: "long", day: "numeric"};
            var display = d.toLocaleDateString(lang, options);

            return display;
        },

        nowNice: function() {
            var d_names = new Array("Sunday", "Monday", "Tuesday",
            "Wednesday", "Thursday", "Friday", "Saturday");

            var m_names = new Array("January", "February", "March", 
            "April", "May", "June", "July", "August", "September", 
            "October", "November", "December");

            var d = new Date();
            var curr_day = d.getDay();
            var curr_date = d.getDate();
            var sup = "";
            if (curr_date == 1 || curr_date == 21 || curr_date ==31)
            {
               sup = "st";
            }
            else if (curr_date == 2 || curr_date == 22)
            {
               sup = "nd";
            }
            else if (curr_date == 3 || curr_date == 23)
            {
               sup = "rd";
            }
            else
            {
               sup = "th";
            }
            
            var curr_month = d.getMonth();
            var curr_year = d.getFullYear();
            var fulldate = d_names[curr_day] + " " + curr_date + sup + ", " + m_names[curr_month] + " " + curr_year //+ "<SUP>" + sup + "</SUP> " 
            var a_p = "";
            var d = new Date();
            var curr_hour = d.getHours();
            
            if (curr_hour < 12)
            {
               a_p = "am";
            }
            else
            {
               a_p = "pm";
            }
            if (curr_hour == 0)
            {
               curr_hour = 12;
            }
            if (curr_hour > 12)
            {
               curr_hour = curr_hour - 12;
            }

            var curr_min = d.getMinutes();

            curr_min = curr_min + "";

            if (curr_min.length == 1)
            {
               curr_min = "0" + curr_min;
            }

            var fulltime = curr_hour + ":" + curr_min + a_p
            var fullstamp = fulltime + ", " + fulldate

            return fullstamp;
        },

        /*
        * pretty date ripped from .....
        *
        * JavaScript Pretty Date
        * Copyright (c) 2008 John Resig (jquery.com)
        * Licensed under the MIT license.
        * 
        * http://ejohn.org/blog/javascript-pretty-date/
        *
        * and modified a bit - i'm sure Resig is fine with it...
        */

        // absDay = true, then show some day names ets, if false then always relative  eg 7 days ago

        absDate: function (time, absDay, dayAccuracy, correctzone) {
            if (typeof correctzone === 'undefined') {
                correctzone = false;   // by defualt we are working in unix time
            }

            var userTime = new Date();
            var time_zone = userTime.getTimezoneOffset();


            var date = time, //new Date((time || "").replace(/-/g,"/").replace(/[TZ]/g," ")),
            diff = (((new Date()).getTime() - date.getTime()) / 1000) + (correctzone ? (time_zone * 60) : 0),
            day_diff = Math.floor(diff / 86400);

            if (diff < -60 ) {
                console.warn("Too far in the future");
                return;
            }

            if (diff < 0) {
                day_diff = 0;
            }

            if (isNaN(day_diff)) {
                console.warn("not a number");
                return;
            }
            
            var dayBit = "Over a month ago";
            // var dayBit = "Just now";

            if (absDay) {
                // if (day_diff < 7) {
                //     dayBit = time.format("dddd");
                // }
                // else if (day_diff < 31) {
                //     dayBit = time.format("dddd dS");
                // }
                // else {
                //     dayBit = time.format("dddd dS mmm");
                // }
            } else {
                if (day_diff < 7) {
                    dayBit = day_diff + " days ago";
                }
                else if (day_diff < 31) {
                    dayBit = Math.ceil( day_diff / 7 ) + " weeks ago";
                }
            }
            
            if (dayAccuracy) {
                return (day_diff === 0 && "Today") || (day_diff === 1 && "Yesterday" ) || dayBit;
            }

            return day_diff === 0 && (
                diff < 60 && "Just now" ||
                diff < 120 && "1 minute ago" ||
                diff < 3600 && Math.floor( diff / 60 ) + " mins ago" ||
                diff < 7200 && "1 hour ago" ||
                diff < 86400 && Math.floor( diff / 3600 ) + " hours ago") ||
            day_diff == 1 && "Yesterday" ||
            dayBit;
        },

        dateFromString: function (when) {
            // strip timezone
            var tzp = when.split('+');
            if (tzp.length > 1) {
                when = tzp[0];
            }
            // strip trailing Z
            tzp = when.split('Z');
            if (tzp.length > 1) {
                when = tzp[0];
            }
            var parts = when.split('T');
            if (parts.length > 1 ) {
                var dps = parts[0].split('-');
                var tps = parts[1].split(':');
                var msp = tps[2].split('.');
                var ms = '0';
                if (msp.length > 1) {
                    ms = msp[1];
                }
                var timeStamp = new Date(Date.UTC(dps[0], dps[1] - 1, dps[2], tps[0], tps[1], msp[0]));
                return timeStamp;
            }
            else {
                console.log("date error " + when);
                return new Date();
            }
        },
    };

    return Utils;
});


if(typeof(String.prototype.trim) === "undefined")
{
    String.prototype.trim = function() 
    {
        return String(this).replace(/^\s+|\s+$/g, '');
    };
}
