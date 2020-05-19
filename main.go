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
const LOG_BUFFER_SIZE = 20

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

func main() {
	go func() {
		// cmd := exec.Command("ping", "-t", "google.com")
		cmd := exec.Command("pm2", "logs")
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
	}()

	go func() {
		for {
			cmd := exec.Command("pm2", "jlist")
			data, err := cmd.Output()
			if err != nil {
				fmt.Println(err)
				continue
			}
			logsChan <- LogData{Type: "stats", Data: string(data), Time: time.Now().UnixNano() / 1e6}
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
					if err := client.WriteJSON(data); err != nil {
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
		for e := logBuffer.Front(); e != nil; e = e.Next() {
			// fmt.Println(e.Value)
			if err := client.WriteJSON(e.Value); err != nil {
				client.Close()
				return
			}
		}
		// if err := client.WriteJSON(LogData{Type: "log", Data: "------------------------------------------", Time: time.Now().UnixNano() / 1e6}); err != nil {
		// 	client.Close()
		// 	return
		// }
		clientsChan <- client
	})))

	if err := http.ListenAndServe(":3030", nil); err != nil {
		fmt.Println(err)
	}
}
