package broadcast

import (
	"sync"
	"time"
)

type MapFunc[In any, Out any] func(In) Out

type BroadcastService[In any, Out any] struct {
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

func NewBroadcastService[In any, Out any](mapFunc MapFunc[In, Out]) *BroadcastService[In, Out] {
	return &BroadcastService[In, Out]{
		channels: make(map[chan<- Out]chan struct{}),
		mapFunc:  mapFunc,
	}
}

func (server *BroadcastService[In, Out]) StartBroadcast(inputChan <-chan In, numWorkers int, timeout time.Duration) error {
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

func (server *BroadcastService[In, Out]) AddClient(c chan<- Out) {
	server.mu.Lock()
	server.channels[c] = make(chan struct{})
	server.mu.Unlock()
}

func (server *BroadcastService[In, Out]) RemoveClient(c chan<- Out) {
	server.mu.Lock()
	done := server.channels[c]
	delete(server.channels, c)
	server.mu.Unlock()
	if done != nil {
		close(done)
	}
}
