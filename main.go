package main

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	Username      string `short:"u" long:"username" description:"BasicAuth username" required:"false" default:""`
	Password      string `short:"p" long:"password" description:"BasicAuth password" required:"false" default:""`
	LogBufferSize int    `short:"l" long:"log-buffer-size" description:"Log buffer size" required:"false" default:"200"`
	Interval      int    `short:"i" long:"interval" description:"PM2 process-list update interval in seconds" required:"false" default:"10"`
}

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

	parser := flags.NewParser(&opts, flags.Default)

	parser.Usage = "[OPTIONS] address"

	args, err := parser.Parse()

	if err != nil {
		os.Exit(1)
	}

	if len(args) == 0 {
		parser.WriteHelp(os.Stdout)
		return
	}

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

	pm2 := NewPM2(time.Duration(opts.Interval)*time.Second, opts.LogBufferSize).Run()
	if err := NewHTTPServer(args[0], opts.Username, opts.Password, pm2).Run(); err != nil {
		log.Fatalln(err)
	}
}
