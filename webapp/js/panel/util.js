"use strict";

export function PrettyDate(time) {
    var date = new Date(time || ""),
        diff = (((new Date()).getTime() - date.getTime()) / 1000),
        day_diff = Math.floor(diff / 86400);
    var year = date.getFullYear(),
        month = date.getMonth()+1,
        day = date.getDate();

    if (isNaN(day_diff) || day_diff < 0 || day_diff >= 31)
        return (
            year.toString()+'-'
            +((month<10) ? '0'+month.toString() : month.toString())+'-'
            +((day<10) ? '0'+day.toString() : day.toString())
        );

    var r =
    ( 
        (
            day_diff == 0 && 
            (
                (diff < 60 && "just now")
                || (diff < 120 && "1 minute ago")
                || (diff < 3600 && Math.floor(diff / 60) + " minutes ago")
                || (diff < 7200 && "1 hour ago")
                || (diff < 86400 && Math.floor(diff / 3600) + " hours ago")
            )
        )
        || (day_diff == 1 && "Yesterday")
        || (day_diff < 7 && day_diff + " days ago")
        || (day_diff < 31 && Math.ceil(day_diff / 7) + " weeks ago")
    );
    return r;
}

export function ToHHMMSS(sec_num) {
    sec_num = Math.floor(sec_num)
    var hours   = Math.floor(sec_num / 3600);
    var minutes = Math.floor((sec_num - (hours * 3600)) / 60);
    var seconds = sec_num - (hours * 3600) - (minutes * 60);

    
    if (minutes < 10) {minutes = "0"+minutes;}
    if (seconds < 10) {seconds = "0"+seconds;}
    if (hours > 0) {
        return hours+':'+minutes+':'+seconds;
    }
    return minutes+':'+seconds;
}
