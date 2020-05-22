package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	//_ "net/http/pprof"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

const USERNAME string = "admin"
const PASSWORD string = "1234"
const LOG_BUFFER_SIZE = 200

type LogData struct {
	Type string
	Data interface{}
	Time int64
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var newClientsChan chan chan LogData = make(chan chan LogData, 100)
var removedClientsChan chan chan LogData = make(chan chan LogData, 100)
var logsChan chan LogData = make(chan LogData, 100)
var logBuffer = list.New()
var stats LogData

func main() {
	go func() {
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
				for logBuffer.Len() >= LOG_BUFFER_SIZE {
					e := logBuffer.Front()
					logBuffer.Remove(e)
				}
				logBuffer.PushBack(logData)
				logsChan <- logData
			}
		}
	}()

	go func() {
		for {
			cmd := exec.Command("pm2", "jlist")
			data, err := cmd.Output()
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
			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		var clients map[chan LogData]bool = make(map[chan LogData]bool)
		for {
			select {
			case client := <-newClientsChan:
				clients[client] = true
				// fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case client := <-removedClientsChan:
				delete(clients, client)
				close(client)
				// fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case data := <-logsChan:
				for client := range clients {
					client <- data
				}
			}
		}
	}()

	http.Handle("/", httpauth.SimpleBasicAuth(USERNAME, PASSWORD)(http.FileServer(http.Dir("./static"))))

	http.Handle("/logs", httpauth.SimpleBasicAuth(USERNAME, PASSWORD)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var client, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		if stats.Type != "" {
			client.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.WriteJSON(stats); err != nil {
				client.Close()
				return
			}
		}
		for e := logBuffer.Front(); e != nil; e = e.Next() {
			client.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.WriteJSON(e.Value); err != nil {
				client.Close()
				return
			}
		}
		clientChan := make(chan LogData, 100)
		fmt.Printf("Client connected from: %s \r\n", client.RemoteAddr().String())
		newClientsChan <- clientChan
		for data := range clientChan {
			client.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.WriteJSON(data); err != nil {
				client.Close()
				// fmt.Printf("Client disconnected from: %s \r\n", client.RemoteAddr().String())
				removedClientsChan <- clientChan
				return
			}
		}
	})))

	if err := http.ListenAndServe(":3030", nil); err != nil {
		fmt.Println(err)
	}
}
