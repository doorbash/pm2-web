package main

import (
	"errors"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/websocket"
)

type HttpServer struct {
	Addr           string
	upgrader       websocket.Upgrader
	options        *Options
	newClients     *chan chan LogData
	removedClients *chan chan LogData
	pm2            *PM2
}

func (h *HttpServer) JsHandler(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("./static/script.js")
	if err != nil {
		fmt.Println(err)
		return
	}
	w.Header().Add("Content-Type", "text/javascript")
	err = templ.Execute(w, h.options)
	if err != nil {
		fmt.Println(err)
	}
}

func (h *HttpServer) LogsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	clientChan := make(chan LogData, 100)
	*h.newClients <- clientChan
	fmt.Printf("Client connected : %s \r\n", conn.RemoteAddr().String())
	for data := range clientChan {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteJSON(data); err != nil {
			conn.Close()
			*h.removedClients <- clientChan
			fmt.Printf("Client disconnected : %s \r\n", conn.RemoteAddr().String())
		}
	}
}

func (h *HttpServer) ActionsHandler(w http.ResponseWriter, r *http.Request) {
	ops, ok := r.URL.Query()["op"]
	if !ok || len(ops[0]) < 1 {
		w.Write([]byte("Url Param 'op' is missing"))
		return
	}

	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		w.Write([]byte("Url Param 'id' is missing"))
		return
	}

	var err error
	switch op := string(ops[0]); op {
	case "start", "stop", "restart":
		err = h.pm2.Action(ids[0], op)
	default:
		err = errors.New("bad op")
	}

	if err != nil {
		w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
	} else {
		w.Write([]byte("ok"))
	}
}

func (h *HttpServer) middleware(f http.HandlerFunc) http.Handler {
	if h.options.Username == "" {
		return f
	} else {
		return httpauth.SimpleBasicAuth(h.options.Username, h.options.Password)(f)
	}
}

func NewHTTPServer(addr string, options *Options, pm2 *PM2, newClients, removedClients *Client) *HttpServer {
	h := &HttpServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Addr:           addr,
		options:        options,
		pm2:            pm2,
		newClients:     newClients,
		removedClients: removedClients,
	}

	http.Handle("/", h.middleware(http.FileServer(http.Dir("./static")).ServeHTTP))
	http.Handle("/script.js", h.middleware(h.JsHandler))
	http.Handle("/logs", h.middleware(h.LogsHandler))
	if options.ActionsEnabled {
		http.Handle("/action", h.middleware(h.ActionsHandler))
	}

	return h
}

func (s *HttpServer) ListenAndServe() error {
	return http.ListenAndServe(s.Addr, nil)
}
