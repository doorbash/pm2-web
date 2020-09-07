package main

import (
	"fmt"
	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func initServer() {
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
		fmt.Printf("Client connected : %s \r\n", client.RemoteAddr().String())
		newClientsChan <- clientChan
		for data := range clientChan {
			client.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.WriteJSON(data); err != nil {
				client.Close()
				fmt.Printf("Client disconnected : %s \r\n", client.RemoteAddr().String())
				removedClientsChan <- clientChan
				return
			}
		}
	})))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil); err != nil {
		fmt.Println(err)
	}
}
