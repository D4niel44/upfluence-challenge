package sse

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	dataPrefix = "data"
)

type ParseFunc[T any] func([]byte) (T, error)

type SSEClient[T any] struct {
	mu        sync.RWMutex
	channels  map[chan<- T]chan struct{}
	parseFunc ParseFunc[T]
}

// DefaultSSEClient is the default SSE client.
var DefaultSSEClient = &SSEClient[[]byte]{
	channels:  make(map[chan<- []byte]chan struct{}),
	parseFunc: func(x []byte) ([]byte, error) { return bytes.TrimSpace(x), nil },
}

type WorkerInput[T any] struct {
	value    T
	outChan  chan<- T
	doneChan <-chan struct{}
}

func worker[T any](in <-chan WorkerInput[T], timeout time.Duration, wg *sync.WaitGroup) {
	for input := range in {
		timeoutSignal := time.After(timeout)
		select {
		case <-timeoutSignal:
			// TODO: Log an error in this case, but dont fail the goroutine.
		case <-input.doneChan:
		case input.outChan <- input.value:
		}
	}
	wg.Done()
}

func startWorkers[T any](numWorkers int, in <-chan WorkerInput[T], timeout time.Duration) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(in, timeout, &wg)
	}
	return &wg
}

func NewSSEClient[T any](parseFunc ParseFunc[T]) *SSEClient[T] {
	return &SSEClient[T]{
		channels:  make(map[chan<- T]chan struct{}),
		parseFunc: parseFunc,
	}
}

func (client *SSEClient[T]) StartListening(url string, numWorkers int, timeout time.Duration) error {
	workers := make(chan WorkerInput[T], 1)
	wg := startWorkers(numWorkers, workers, timeout)
	defer func() {
		close(workers)
		wg.Wait()
		client.mu.Lock()
		for k := range client.channels {
			close(k)
			delete(client.channels, k)
		}
		client.mu.Unlock()
	}()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(res.Body)
	defer res.Body.Close()

	sep := []byte{':'}
	for {
		readBytes, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}

		data := bytes.SplitN(readBytes, sep, 2)
		if string(data[0]) == dataPrefix {
			parsed, err := client.parseFunc(bytes.TrimSpace(data[1]))
			if err != nil {
				return err
			}
			client.mu.RLock()
			for out, done := range client.channels {
				workers <- WorkerInput[T]{
					value:    parsed,
					outChan:  out,
					doneChan: done,
				}
			}
			client.mu.RUnlock()
		}

		if err == io.EOF {
			break
		}
	}
	return nil
}

func (client *SSEClient[T]) RegisterChannel(c chan<- T) {
	client.mu.Lock()
	client.channels[c] = make(chan struct{})
	client.mu.Unlock()
}

func (client *SSEClient[T]) UnregisterChannel(c chan<- T) {
	client.mu.Lock()
	done := client.channels[c]
	delete(client.channels, c)
	client.mu.Unlock()
	if done != nil {
		close(done)
	}
}

func StartListening(url string, numWorkers int, timeout time.Duration) error {
	return DefaultSSEClient.StartListening(url, numWorkers, timeout)
}

func RegisterChannel(c chan<- []byte) {
	DefaultSSEClient.RegisterChannel(c)
}

func UnregisterChannel(c chan<- []byte) {
	DefaultSSEClient.UnregisterChannel(c)
}
