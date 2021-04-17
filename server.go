package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

type HttpServer struct {
	Addr     string
	upgrader websocket.Upgrader
}

func NewServer(addr, username, passwrod string) *HttpServer {
	return (&HttpServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Addr: addr,
	}).init(username, passwrod)
}

func (s *HttpServer) init(username, password string) *HttpServer {
	staticHandler := http.FileServer(http.Dir("./static"))

	logsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var client, err = s.upgrader.Upgrade(w, r, nil)
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
	})

	if username == "" {
		http.Handle("/", staticHandler)
		http.Handle("/logs", logsHandler)
	} else {
		http.Handle("/", httpauth.SimpleBasicAuth(username, password)(staticHandler))
		http.Handle("/logs", httpauth.SimpleBasicAuth(username, password)(logsHandler))
	}

	return s
}

func (s *HttpServer) Run() error {
	return http.ListenAndServe(s.Addr, nil)
}
