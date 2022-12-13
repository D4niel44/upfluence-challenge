// Package broadcast provides an utility to broadcast the values received to multiple clients.
package broadcast

import (
	"sync"
	"time"
)

// MapFunc can be used to modify the values of the input channel,
// before broadcasting them to each client.
type MapFunc[In any, Out any] func(In) Out

// Service forwards the values received through an input channel,
// to many output channels. Each value received is first maped using the
// given MapFunc, before forwarding them.
type Service[In any, Out any] struct {
	mu       sync.RWMutex
	channels map[chan<- Out]chan struct{}
	mapFunc  MapFunc[In, Out]
}

type workerInput[T any] struct {
	value    T
	outChan  chan<- T
	doneChan <-chan struct{}
}

func worker[T any](in <-chan workerInput[T], timeout time.Duration, wg *sync.WaitGroup) {
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

func startWorkers[T any](numWorkers int, in <-chan workerInput[T], timeout time.Duration) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(in, timeout, &wg)
	}
	return &wg
}

// NewBroadcastService creates a new BroadCastService, and initializes it.
func NewBroadcastService[In any, Out any](mapFunc MapFunc[In, Out]) *Service[In, Out] {
	return &Service[In, Out]{
		channels: make(map[chan<- Out]chan struct{}),
		mapFunc:  mapFunc,
	}
}

// StartBroadcast starts broadcasting the values received on the given channel to all registered clients.
// The parameters numWorkers and timeout can be used to control how many worker goroutines are used to
// forward the values to the clients, and to set a tiemout after which a worker thread will stop trying
// to send a value to a client.
func (server *Service[In, Out]) StartBroadcast(inputChan <-chan In, numWorkers int, timeout time.Duration) error {
	workers := make(chan workerInput[Out], 1)
	wg := startWorkers(numWorkers, workers, timeout)
	defer func() {
		close(workers)
		wg.Wait()
		server.mu.Lock()
		for k := range server.channels {
			close(k)
			delete(server.channels, k)
		}
		server.mu.Unlock()
	}()

	for inputValue := range inputChan {
		outputValue := server.mapFunc(inputValue)
		server.mu.RLock()
		for out, done := range server.channels {
			workers <- workerInput[Out]{
				value:    outputValue,
				outChan:  out,
				doneChan: done,
			}
		}
		server.mu.RUnlock()
	}
	return nil
}

// AddClient adds a new client, to start receiving messages through
// the given channel. This method blocks until the client can be added.
func (server *Service[In, Out]) AddClient(c chan<- Out) {
	server.mu.Lock()
	server.channels[c] = make(chan struct{})
	server.mu.Unlock()
}

// RemoveClient removes a client from receiving messages through the channel.
// This method should always be called when the client no longer wants to receive
// messages.
// Calling this method with an already removed client is a no op.
func (server *Service[In, Out]) RemoveClient(c chan<- Out) {
	server.mu.Lock()
	done := server.channels[c]
	delete(server.channels, c)
	server.mu.Unlock()
	if done != nil {
		close(done)
	}
}
