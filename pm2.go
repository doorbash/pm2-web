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

type PM2 struct {
	Interval      time.Duration
	LogBufferSize int
}

func NewPm2(interval time.Duration, logBufferSize int) *PM2 {
	return &PM2{
		Interval:      interval,
		LogBufferSize: logBufferSize,
	}
}

func (p *PM2) Run() {
	go p.logs()
	go p.jlist()
}

func (p *PM2) logs() {
	for {
		// cmd := exec.Command("ping", "google.com", "-c", "10")
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
			// fmt.Printf("data=%s\n", data) // timestamp app id type message
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
			var jM map[string]string = make(map[string]string)
			jM["time"] = fmt.Sprintf("%s%c%s", data[10:20], ' ', data[21:idx1])
			jM["app"] = data[idx1+5 : idx2]
			jM["id"] = data[idx2+4 : idx3]
			jM["type"] = data[idx3+6 : idx4]
			jM["message"] = data[idx4+9:]
			logData := LogData{Type: "log", Data: jM, Time: time.Now().UnixNano() / 1e6}
			for logBuffer.Len() >= p.LogBufferSize {
				e := logBuffer.Front()
				logBuffer.Remove(e)
			}
			logBuffer.PushBack(logData)
			logsChan <- logData
		}
	}
}

func (p *PM2) jlist() {
	for {
		cmd := exec.Command("pm2", "jlist")
		data, err := cmd.Output()
		// fmt.Println(string(data))
		if err != nil {
			log.Fatal(err)
		}
		var sObject []interface{}
		json.Unmarshal(data, &sObject)
		var oObject []interface{} = make([]interface{}, len(sObject))
		for i := range sObject {
			oObject[i] = make(map[string]interface{})
			oObject[i].(map[string]interface{})["name"] = sObject[i].(map[string]interface{})["name"]
			oObject[i].(map[string]interface{})["id"] = sObject[i].(map[string]interface{})["pm_id"]
			oObject[i].(map[string]interface{})["pid"] = sObject[i].(map[string]interface{})["pid"]
			oObject[i].(map[string]interface{})["uptime"] = sObject[i].(map[string]interface{})["pm2_env"].(map[string]interface{})["pm_uptime"]
			oObject[i].(map[string]interface{})["status"] = sObject[i].(map[string]interface{})["pm2_env"].(map[string]interface{})["status"]
			oObject[i].(map[string]interface{})["restart"] = sObject[i].(map[string]interface{})["pm2_env"].(map[string]interface{})["restart_time"]
			oObject[i].(map[string]interface{})["user"] = sObject[i].(map[string]interface{})["pm2_env"].(map[string]interface{})["username"]
			oObject[i].(map[string]interface{})["cpu"] = sObject[i].(map[string]interface{})["monit"].(map[string]interface{})["cpu"]
			oObject[i].(map[string]interface{})["mem"] = sObject[i].(map[string]interface{})["monit"].(map[string]interface{})["memory"]
		}
		stats = LogData{Type: "stats", Data: oObject, Time: time.Now().UnixNano() / 1e6}
		logsChan <- stats
		time.Sleep(p.Interval)
	}
}
