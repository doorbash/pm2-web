package main

import (
	"bufio"
	"container/list"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

const USERNAME string = "admin"
const PASSWORD string = "1234"
const LOG_BUFFER_SIZE = 200

type LogData struct {
	Type string
	Data string
	Time int64
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clientsChan chan *websocket.Conn = make(chan *websocket.Conn)
var logsChan chan LogData = make(chan LogData)
var logBuffer = list.New()
var stats LogData

func main() {
	go func() {
		for {
			// cmd := exec.Command("ping", "google.com", "-c", "10")
			cmd := exec.Command("pm2", "logs", "--json")
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
				logData := LogData{Type: "log", Data: data, Time: time.Now().UnixNano() / 1e6}
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
			stats = LogData{Type: "stats", Data: string(data), Time: time.Now().UnixNano() / 1e6}
			logsChan <- stats
			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		var clients map[*websocket.Conn]bool = make(map[*websocket.Conn]bool)
		var clientsRemoveList []*websocket.Conn
		for {
			select {
			case client := <-clientsChan:
				clients[client] = true
			case data := <-logsChan:
				for client := range clients {
					client.SetWriteDeadline(time.Now().Add(5 * time.Second))
					if err := client.WriteJSON(data); err != nil {
						fmt.Println(err)
						client.Close()
						clientsRemoveList = append(clientsRemoveList, client)
						continue
					}
				}
				if len(clientsRemoveList) > 0 {
					for _, clientToRemove := range clientsRemoveList {
						delete(clients, clientToRemove)
					}
					clientsRemoveList = nil
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
			if err := client.WriteJSON(stats); err != nil {
				client.Close()
				return
			}
		}
		for e := logBuffer.Front(); e != nil; e = e.Next() {
			// fmt.Println(e.Value)
			if err := client.WriteJSON(e.Value); err != nil {
				client.Close()
				return
			}
		}
		clientsChan <- client
	})))

	if err := http.ListenAndServe(":3030", nil); err != nil {
		fmt.Println(err)
	}
}
