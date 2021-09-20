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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
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
	q := r.URL.Query()
	op := q.Get("op")
	id := q.Get("id")

	if op == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Url Param 'op' is missing"))
		return
	}

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Url Param 'id' is missing"))
		return
	}

	var err error
	switch op {
	case "start", "stop", "restart":
		err = h.pm2.Action(id, op)
	default:
		err = errors.New("bad op")
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error: %s\n", err.Error())))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	}
}

func (h *HttpServer) middleware(f http.HandlerFunc) http.Handler {
	if h.options.Username == "" {
		return f
	} else {
		return httpauth.SimpleBasicAuth(h.options.Username, h.options.Password)(f)
	}
}

func NewHTTPServer(
	addr string,
	options *Options,
	pm2 *PM2,
	newClients,
	removedClients *chan chan LogData,
) *HttpServer {
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
