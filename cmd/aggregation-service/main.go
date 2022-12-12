package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/D4niel44/upfluence-challenge/internal/aggregations"
)

const (
	port = ":8080"
)

var sseService = &SSEService{
	url:        "https://stream.upfluence.co/stream",
	numWorkers: 10000,
	timeout:    5 * time.Second,
}

func getAnalysis(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got request")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	durationString := r.URL.Query().Get("duration")
	if durationString == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dimension := r.URL.Query().Get("dimension")
	if dimension == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	duration, err := time.ParseDuration(durationString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	in := make(chan map[string]interface{})
	broadcastService.AddClient(in)
	defer broadcastService.RemoveClient(in)
	aggregatorState := aggregations.InitialState(dimension)
	timeout := time.After(duration)
loop:
	for {
		select {
		case <-timeout:
			break loop
		case post := <-in:
			err := aggregatorState.Aggregate(post)
			if err != nil {
				WarningLogger.Println(err)
				WarningLogger.Println(post)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	if aggregatorState.TotalPosts == 0 {
		aggregatorState.MinTimestamp = 0
		aggregatorState.MaxTimestamp = 0
	}
	w.Write(EncodeResponse(aggregatorState))
}

func EncodeResponse(aggregation *aggregations.AggregatorState) []byte {
	response := make(map[string]interface{})
	response["total_posts"] = aggregation.TotalPosts
	response["minimum_timestamp"] = aggregation.MinTimestamp
	response["maximum_timestamp"] = aggregation.MaxTimestamp
	response["avg_"+aggregation.Dimension] = aggregation.DimensionAvg
	responseBlob, _ := json.Marshal(response)
	return responseBlob
}

func main() {
	sseService.startSSEService()

	http.HandleFunc("/analysis", getAnalysis)
	ErrorLogger.Fatal(http.ListenAndServe(port, nil))
}
