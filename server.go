package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

type HttpServer struct {
	Addr     string
	upgrader websocket.Upgrader
}

func NewHTTPServer(addr string, options *Options, pm2 *PM2, newClientsChan *chan chan LogData, removedClientsChan *chan chan LogData) *HttpServer {
	s := &HttpServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Addr: addr,
	}

	staticHandler := http.FileServer(http.Dir("./static"))

	jsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templ, err := template.ParseFiles("./static/script.js")
		if err != nil {
			fmt.Println(err)
			return
		}
		w.Header().Add("Content-Type", "text/javascript")
		err = templ.Execute(w, options)
		if err != nil {
			fmt.Println(err)
		}
	})

	logsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		clientChan := make(chan LogData, 100)
		*newClientsChan <- clientChan
		fmt.Printf("Client connected : %s \r\n", conn.RemoteAddr().String())
		for data := range clientChan {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(data); err != nil {
				conn.Close()
				*removedClientsChan <- clientChan
				fmt.Printf("Client disconnected : %s \r\n", conn.RemoteAddr().String())
			}
		}
	})

	actionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			err := pm2.StartProcess(ids[0])
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
			err := pm2.StopProcess(ids[0])
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
			err := pm2.RestartProcess(ids[0])
			if err != nil {
				w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
			} else {
				w.Write([]byte("ok"))
			}

		default:
			w.Write([]byte("bad op"))
		}
	})

	if options.Username == "" {
		http.Handle("/", staticHandler)
		http.Handle("/script.js", jsHandler)
		http.Handle("/logs", logsHandler)
		if options.ActionsEnabled {
			http.Handle("/action", actionHandler)
		}
	} else {
		http.Handle("/", httpauth.SimpleBasicAuth(options.Username, options.Password)(staticHandler))
		http.Handle("/script.js", httpauth.SimpleBasicAuth(options.Username, options.Password)(jsHandler))
		http.Handle("/logs", httpauth.SimpleBasicAuth(options.Username, options.Password)(logsHandler))
		if options.ActionsEnabled {
			http.Handle("/action", httpauth.SimpleBasicAuth(options.Username, options.Password)(actionHandler))
		}
	}

	return s
}

func (s *HttpServer) Run() error {
	return http.ListenAndServe(s.Addr, nil)
}
