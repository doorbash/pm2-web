package main

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Username       string `short:"u" long:"username" description:"BasicAuth username" required:"false" default:""`
	Password       string `short:"p" long:"password" description:"BasicAuth password" required:"false" default:""`
	LogBufferSize  int    `short:"l" long:"log-buffer-size" description:"Log buffer size" required:"false" default:"200"`
	Interval       int    `short:"i" long:"interval" description:"PM2 process-list update interval in seconds" required:"false" default:"10"`
	TimeEnabled    bool   `long:"time" description:"Show log time" required:"false"`
	AppIdEnabled   bool   `long:"app-id" description:"Show app id" required:"false"`
	AppNameEnabled bool   `long:"app-name" description:"Show app name" required:"false"`
	ActionsEnabled bool   `long:"actions" description:"Show start, stop and restart buttons"`
}

func (o *Options) Valid() bool {
	if o.LogBufferSize < 0 {
		return false
	}
	if opts.Interval < 0 {
		return false
	}
	return true
}

var opts Options

var newClientsChan chan chan LogData = make(chan chan LogData, 100)
var removedClientsChan chan chan LogData = make(chan chan LogData, 100)
var logsChan chan LogData = make(chan LogData, 100)
var statsChan chan LogData = make(chan LogData)
var logBuffer *list.List = list.New()
var stats LogData

func main() {
	parser := flags.NewParser(&opts, flags.Default)

	parser.Usage = "[OPTIONS] address"

	args, err := parser.Parse()

	if err != nil {
		log.Fatalln(err)
	}

	if len(args) == 0 {
		parser.WriteHelp(os.Stdout)
		return
	}

	if !opts.Valid() {
		log.Fatalln("bad options")
	}

	go func() {
		var clients map[chan LogData]bool = make(map[chan LogData]bool)
		for {
			select {
			case client := <-newClientsChan:
				clients[client] = true
				if stats.Type != "" {
					select {
					case client <- stats:
					default:
					}
				}
				for e := logBuffer.Front(); e != nil; e = e.Next() {
					select {
					case client <- e.Value.(LogData):
					default:
					}
				}
				fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case client := <-removedClientsChan:
				delete(clients, client)
				close(client)
				fmt.Printf("Num connected clients : %d \r\n", len(clients))
			case logData := <-logsChan:
				for logBuffer.Len() >= opts.LogBufferSize {
					logBuffer.Remove(logBuffer.Front())
				}
				logBuffer.PushBack(logData)
				for client := range clients {
					select {
					case client <- logData:
					default:
					}
				}
			case stats = <-statsChan:
				for client := range clients {
					select {
					case client <- stats:
					default:
					}
				}
			}
		}
	}()

	pm2 := NewPM2(time.Duration(opts.Interval)*time.Second, &statsChan, &logsChan).Start()

	if err := NewHTTPServer(args[0], &opts, pm2, &newClientsChan, &removedClientsChan).Run(); err != nil {
		log.Fatalln(err)
	}
}
