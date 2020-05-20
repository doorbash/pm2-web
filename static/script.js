let host = window.document.location.host.replace(/:.*/, '');
let socket = new WebSocket(location.protocol.replace("http", "ws") + "//" + host + (location.port ? ':' + location.port : '') + "/logs")
document.title = "PM2 | " + host
let statsHeight = 0;

function updateUI() {
    document.getElementById("logs").style.top = (document.getElementById("stats").offsetHeight + 10) + "px";
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
    // console.log(data)
    if (data.Type == "log") {
        let div = document.getElementById("logs");
        let text = div.innerHTML
        // let logs = text.split("<br>");
        // let numLines = logs.length;
        // if (numLines >= 50) {
        //     div.innerHTML = logs.slice(1).join("<br>")
        // }
        let log = JSON.parse(data.Data);
        div.innerHTML += "<span style=\"color: lime\">" + log.app_name + "</span>" + " > " + log.message + "<br>"
        div.scrollTop = div.scrollHeight;
    } else if (data.Type == "stats") {
        let stats = JSON.parse(data.Data)
        // console.log(stats)
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
            let uptime = Math.floor((data.Time - stats[i].pm2_env.pm_uptime) / 1000);
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

            let status = stats[i].pm2_env.status;

            txt += "<tr>"
            txt += "<td class=\"table_title\">" + stats[i].name + "</td>"
            txt += "<td>" + stats[i].pm_id + "</td>"
            txt += "<td>" + stats[i].pid + "</td>"
            txt += "<td style=\"color: " + (status == 'online' ? "#00ff00" : "#ff0000") + "\">" + status + "</td>"
            txt += "<td>" + stats[i].pm2_env.restart_time + "</td>"
            txt += "<td>" + uptime_txt + "</td>"
            txt += "<td>" + stats[i].monit.cpu + "%</td>"
            txt += "<td>" + (stats[i].monit.memory / (1024 * 1024)).toFixed(1) + " MB</td>"
            txt += "<td>" + stats[i].pm2_env.username + "</td>"
            txt += "</tr>"
        }
        txt += "</table>"
        document.getElementById("stats").innerHTML = txt;
        updateUI();
    }
}

window.onresize = function() {
    console.log("window resize")
    updateUI();
}