const SHOW_TIME = false;
const SHOW_ID = false;
const SHOW_APP_NAME = true;

let host = window.document.location.host.replace(/:.*/, '');
let socket = new WebSocket(location.protocol.replace("http", "ws") + "//" + host + (location.port ? ':' + location.port : '') + "/logs")
document.title = "PM2 | " + host
let statsHeight = 0;

function getSelectedText() {
    var text = "";
    if (typeof window.getSelection != "undefined") {
        text = window.getSelection().toString();
    } else if (typeof document.selection != "undefined" && document.selection.type == "Text") {
        text = document.selection.createRange().text;
    }
    return text;
}

socket.onopen = () => {
    console.log("ws open")
}

socket.onclose = event => {
    console.log("ws close", event)
}

socket.onerror = event => {
    console.log("ws error", event)
}

socket.onmessage = message => {
    let data = JSON.parse(message.data);
    if (data.Type == "log") {
        let log = data.Data;
        if (log.type !== "out" && log.type !== "error") return;
        let div = document.getElementById("logs");
        let lines = div.getElementsByClassName('log')
        while (lines.length > 999) lines[0].remove();
        let p = document.createElement("p");
        p.setAttribute("class", "log");
        let span = document.createElement("span");
        span.setAttribute("style", "color: " + (log.type == "out" ? "#00bb00" : "#800000" + ";"));
        if(SHOW_TIME) span.appendChild(document.createTextNode("[" + new Date(log.time).toLocaleString() + "] "));
        if(SHOW_ID) span.appendChild(document.createTextNode(log.id + " "));
        if(SHOW_APP_NAME) span.appendChild(document.createTextNode(log.app + " "));
        p.appendChild(span);
        p.appendChild(document.createTextNode("> " + log.message));
        let isScrolledToBottom = div.scrollHeight - div.clientHeight <= div.scrollTop + div.offsetHeight * 0.25
        div.appendChild(p);
        if (isScrolledToBottom && !getSelectedText()) {
            div.scrollTop = div.scrollHeight - div.clientHeight
        }
    } else if (data.Type == "stats") {
        let stats = data.Data;
        let txt = "<table>"
        txt += "<tr class=\"table_title\">";
        txt += "<td>App name</td>"
        txt += "<td>id</td>"
        txt += "<td>pid</td>"
        txt += "<td>status</td>"
        txt += "<td>restart</td>"
        txt += "<td>uptime</td>"
        txt += "<td>cpu</td>"
        txt += "<td>mem</td>"
        txt += "<td>user</td>"
        txt += "</tr>"
        for (var i in stats) {
            let uptime = Math.floor((data.Time - stats[i].uptime) / 1000);
            let uptime_txt = uptime % 60 + "s";
            uptime = Math.floor(uptime / 60);
            if (uptime > 0) {
                uptime_txt = uptime % 60 + "m"
                uptime = Math.floor(uptime / 60);
                if (uptime > 0) {
                    uptime_txt = uptime % 24 + "h"
                    uptime = Math.floor(uptime / 24);
                    if (uptime > 0) {
                        uptime_txt = uptime + "d"
                    }
                }
            }

            let status = stats[i].status;

            txt += "<tr>"
            txt += "<td class=\"table_title\">" + stats[i].name + "</td>"
            txt += "<td>" + stats[i].id + "</td>"
            txt += "<td>" + stats[i].pid + "</td>"
            txt += "<td style=\"color: " + (status == 'online' ? "#00ff00" : "#ff0000") + ";\">" + status + "</td>"
            txt += "<td>" + stats[i].restart + "</td>"
            txt += "<td>" + uptime_txt + "</td>"
            txt += "<td>" + stats[i].cpu + "%</td>"
            txt += "<td>" + (stats[i].mem / (1024 * 1024)).toFixed(1) + " MB</td>"
            txt += "<td>" + stats[i].user + "</td>"
            txt += "</tr>"
        }
        txt += "</table>"
        document.getElementById("stats").innerHTML = txt;
        let div = document.getElementById("logs");
        statsHeight = document.getElementById("stats").offsetHeight;
        div.style.top = (statsHeight + 10) + "px";
        let isScrolledToBottom = div.scrollHeight - div.clientHeight <= div.scrollTop + div.offsetHeight * 0.25
        if (isScrolledToBottom && !getSelectedText()) {
            div.scrollTop = div.scrollHeight - div.clientHeight
        }
    }
}

window.onresize = function () {
    let div = document.getElementById("logs");
    statsHeight = document.getElementById("stats").offsetHeight;
    div.style.top = (statsHeight + 10) + "px";
    if (!getSelectedText()) {
        div.scrollTop = div.scrollHeight - div.clientHeight
    }
}