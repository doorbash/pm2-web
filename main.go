package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var logchan chan []byte = make(chan []byte)

func runWs() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	})

	http.HandleFunc("/pm2logs", func(w http.ResponseWriter, r *http.Request) {
		var conn, _ = upgrader.Upgrade(w, r, nil)
		go func(conn *websocket.Conn) {
			for {
				conn.WriteMessage(websocket.TextMessage, <-logchan)
			}
		}(conn)
	})
	err := http.ListenAndServe(":3030", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func runPm2Log() {
	cmd := exec.Command("pm2", "logs", "--json")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			logchan <- scanner.Bytes()
		}
	}()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// if err := cmd.Wait(); err != nil {
	// 	log.Fatal(err)
	// }
}

func main() {
	go runPm2Log()
	runWs()
}
