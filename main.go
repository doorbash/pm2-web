package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

const USERNAME string = "admin"
const PASSWORD string = "1234"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clientsChan chan *websocket.Conn = make(chan *websocket.Conn)
var logsChan chan []byte = make(chan []byte)

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
			data := scanner.Bytes()
			// fmt.Println(string(data))
			logsChan <- data
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
					if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
						client.Close()
						clientsRemoveList = append(clientsRemoveList, client)
						continue
					}
				}
				if len(clientsRemoveList) > 0 {
					for _, clientToRemove := range clientsRemoveList {
						// fmt.Println("Removing a client from the list...")
						delete(clients, clientToRemove)
					}
					clientsRemoveList = nil
				}
			}

		}
	}()

	http.Handle("/", httpauth.SimpleBasicAuth(USERNAME, PASSWORD)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	})))

	http.Handle("/logs", httpauth.SimpleBasicAuth(USERNAME, PASSWORD)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		clientsChan <- conn
	})))

	if err := http.ListenAndServe(":3030", nil); err != nil {
		fmt.Println(err)
	}
}
