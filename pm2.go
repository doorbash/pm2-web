package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type JsonObject = map[string]interface{}

type PM2 struct {
	Interval  time.Duration
	statsChan *chan LogData
	logsChan  *chan LogData
}

func NewPM2(interval time.Duration, statsChan *chan LogData, logsChan *chan LogData) *PM2 {
	return &PM2{
		Interval:  interval,
		statsChan: statsChan,
		logsChan:  logsChan,
	}
}

func (p *PM2) Start() *PM2 {
	go p.logs()
	go p.jlist()
	return p
}

func (p *PM2) logs() {
	for {
		cmd := exec.Command("pm2", "logs", "--format", "--timestamp")
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(cmdReader)
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		for scanner.Scan() {
			data := scanner.Text()
			if !strings.HasPrefix(data, "timestamp=") {
				continue
			}
			idx1 := strings.Index(data, " ")
			if idx1 < 0 || !strings.HasPrefix(data[idx1+1:], "app=") {
				continue
			}
			idx2 := idx1 + strings.Index(data[idx1+1:], " ") + 1
			if idx2 < 0 || !strings.HasPrefix(data[idx2+1:], "id=") {
				continue
			}
			idx3 := idx2 + strings.Index(data[idx2+1:], " ") + 1
			if idx3 < 0 || !strings.HasPrefix(data[idx3+1:], "type=") {
				continue
			}
			idx4 := idx3 + strings.Index(data[idx3+1:], " ") + 1
			if idx4 < 0 || !strings.HasPrefix(data[idx4+1:], "message=") {
				continue
			}
			var jM JsonObject = make(JsonObject)
			jM["time"] = fmt.Sprintf("%s%c%s", data[10:20], ' ', data[21:idx1])
			jM["app"] = data[idx1+5 : idx2]
			jM["id"] = data[idx2+4 : idx3]
			jM["type"] = data[idx3+6 : idx4]
			jM["message"] = data[idx4+9:]
			logData := LogData{Type: "log", Data: jM, Time: time.Now().UnixNano() / 1e6}
			*p.logsChan <- logData
		}
	}
}

func (p *PM2) getJlist() {
	cmd := exec.Command("pm2", "jlist")
	data, err := cmd.Output()
	// fmt.Println(string(data))
	if err != nil {
		log.Fatal(err)
	}
	var sObject []JsonObject
	json.Unmarshal(data, &sObject)
	var oObject []JsonObject = make([]JsonObject, len(sObject))
	for i := range sObject {
		oObject[i] = make(JsonObject)
		oObject[i]["name"] = sObject[i]["name"]
		oObject[i]["id"] = sObject[i]["pm_id"]
		oObject[i]["pid"] = sObject[i]["pid"]
		oObject[i]["uptime"] = sObject[i]["pm2_env"].(JsonObject)["pm_uptime"]
		oObject[i]["status"] = sObject[i]["pm2_env"].(JsonObject)["status"]
		oObject[i]["restart"] = sObject[i]["pm2_env"].(JsonObject)["restart_time"]
		oObject[i]["user"] = sObject[i]["pm2_env"].(JsonObject)["username"]
		oObject[i]["cpu"] = sObject[i]["monit"].(JsonObject)["cpu"]
		oObject[i]["mem"] = sObject[i]["monit"].(JsonObject)["memory"]
	}
	select {
	case *p.statsChan <- LogData{Type: "stats", Data: oObject, Time: time.Now().UnixNano() / 1e6}:
	case <-time.After(3 * time.Second):
	}
}

func (p *PM2) jlist() {
	for {
		p.getJlist()
		time.Sleep(p.Interval)
	}
}

func (p *PM2) Action(id string, command string) error {
	cmd := exec.Command("pm2", command, id)
	_, err := cmd.Output()
	if err == nil {
		go p.getJlist()
	}
	return err
}
