package main

import (
	"container/list"
	"fmt"
	//_ "net/http/pprof"
)

const USERNAME string = "admin"
const PASSWORD string = "1234"
const PORT int = 3030
const LOG_BUFFER_SIZE = 200

type LogData struct {
	Type string
	Data interface{}
	Time int64
}

var newClientsChan chan chan LogData = make(chan chan LogData, 100)
var removedClientsChan chan chan LogData = make(chan chan LogData, 100)
var logsChan chan LogData = make(chan LogData, 100)
var logBuffer = list.New()
var stats LogData

func main() {
	go logs()

	go jlist()

	go func() {
		var clients map[chan LogData]bool = make(map[chan LogData]bool)
		for {
			select {
			case client := <-newClientsChan:
				clients[client] = true
				fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case client := <-removedClientsChan:
				delete(clients, client)
				close(client)
				fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case data := <-logsChan:
				for client := range clients {
					client <- data
				}
			}
		}
	}()

	initServer()
}
