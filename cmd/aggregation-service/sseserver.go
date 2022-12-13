package main

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/D4niel44/upfluence-challenge/internal/broadcast"
	"github.com/D4niel44/upfluence-challenge/internal/sse"
)

type SSEService struct {
	url        string
	numWorkers int
	timeout    time.Duration
}

func parseJSONEvent(eventBlob []byte) (eventData map[string]interface{}) {
	d := json.NewDecoder(bytes.NewReader(eventBlob))
	d.UseNumber()
	if err := d.Decode(&eventData); err != nil {
		WarningLogger.Println("Error parsing json event: ", err)
	}
	return
}

var broadcastService = broadcast.NewBroadcastService(parseJSONEvent)

func (config *SSEService) startSSEService() {
	c := make(chan []byte, 1)

	go func() {
		for {
			err := sse.StartListening(config.url, c)
			if err != nil {
				ErrorLogger.Println(err)
			} else {
				ErrorLogger.Println("Conection to sse server closed.")
			}
			time.Sleep(time.Second)
		}
	}()

	go func() {
		for {
			err := broadcastService.StartBroadcast(c, config.numWorkers, config.timeout)
			if err != nil {
				ErrorLogger.Println(err)
			} else {
				ErrorLogger.Println("Broadcast server crashed.")
			}
			time.Sleep(time.Second)
		}
	}()
}
