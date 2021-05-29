package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

type HttpServer struct {
	Addr     string
	upgrader websocket.Upgrader
}

func NewHTTPServer(addr, username, passwrod string, pm2 *PM2) *HttpServer {
	return (&HttpServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Addr: addr,
	}).init(username, passwrod, pm2)
}

func (s *HttpServer) init(username, password string, pm2 *PM2) *HttpServer {
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

	commandHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ops, ok := r.URL.Query()["op"]
		if !ok || len(ops[0]) < 1 {
			log.Println("Url Param 'op' is missing")
			return
		}

		op := string(ops[0])

		switch op {
		case "start":
			ids, ok := r.URL.Query()["id"]
			if !ok || len(ids[0]) < 1 {
				w.Write([]byte("Url Param 'id' is missing"))
				return
			}
			err := pm2.Start(ids[0])
			if err != nil {
				w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
			} else {
				w.Write([]byte("ok"))
			}

		case "stop":
			ids, ok := r.URL.Query()["id"]
			if !ok || len(ids[0]) < 1 {
				w.Write([]byte("Url Param 'id' is missing"))
				return
			}
			err := pm2.Stop(ids[0])
			if err != nil {
				w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
			} else {
				w.Write([]byte("ok"))
			}

		case "restart":
			ids, ok := r.URL.Query()["id"]
			if !ok || len(ids[0]) < 1 {
				w.Write([]byte("Url Param 'id' is missing"))
				return
			}
			err := pm2.Restart(ids[0])
			if err != nil {
				w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
			} else {
				w.Write([]byte("ok"))
			}

		default:
			log.Printf("op = %s is not supported " + op)
			w.Write([]byte("bad op"))
		}
	})

	if username == "" {
		http.Handle("/", staticHandler)
		http.Handle("/logs", logsHandler)
		http.Handle("/command", commandHandler)
	} else {
		http.Handle("/", httpauth.SimpleBasicAuth(username, password)(staticHandler))
		http.Handle("/logs", httpauth.SimpleBasicAuth(username, password)(logsHandler))
		http.Handle("/command", httpauth.SimpleBasicAuth(username, password)(commandHandler))
	}

	return s
}

func (s *HttpServer) Run() error {
	return http.ListenAndServe(s.Addr, nil)
}
