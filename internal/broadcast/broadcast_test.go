package broadcast

import (
	"sync"
	"testing"
	"time"
)

var testData = []string{
	"123",
	"4",
	"789abrhv",
	"100",
	"10000: {}",
	"de",
	"assvv",
	"aaa",
	"1fh",
	"asb",
}

func assertEqual(t *testing.T, expected string, actual string) {
	if actual != expected {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}

func initBroadcastService() *BroadcastService[string, string] {
	return NewBroadcastService(func(x string) (string, error) { return x, nil })
}

func sendData() chan string {
	c := make(chan string)
	go func() {
		for _, s := range testData {
			c <- s
		}
		close(c)
	}()
	return c
}

func TestNewBroadcastService(t *testing.T) {
	server := initBroadcastService()
	if server.channels == nil {
		t.Fatal("Map was not initialized")
	}
	if server.mapFunc == nil {
		t.Fatal("Map function was not initialized")
	}
}

func TestBroadcastServiceOneChannel(t *testing.T) {
	server := initBroadcastService()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan string)
	server.AddClient(c)
	defer server.RemoveClient(c)
	go func() {
		defer wg.Done()
		for _, expected := range testData {
			actual := string(<-c)
			assertEqual(t, expected, actual)
		}
	}()

	if err := server.StartBroadcast(sendData(), 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBroadcastServiceMultipleChannels(t *testing.T) {
	server := initBroadcastService()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < len(testData); i++ {
		c := make(chan string)
		server.AddClient(c)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < i; j++ {
				actual := string(<-c)
				assertEqual(t, testData[j], actual)
			}
			server.RemoveClient(c)
		}(i)
	}
	if err := server.StartBroadcast(sendData(), 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBroadcastServiceMultipleWorkers(t *testing.T) {
	server := initBroadcastService()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < len(testData); i++ {
		c := make(chan string)
		server.AddClient(c)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < i; j++ {
				<-c
			}
			server.RemoveClient(c)
		}(i)
	}

	if err := server.StartBroadcast(sendData(), 10, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBroadcastServiceTimeout(t *testing.T) {
	server := initBroadcastService()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < 10; i++ {
		c := make(chan string)
		server.AddClient(c)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Second)
			server.RemoveClient(c)
		}()
	}

	c := make(chan string)
	server.AddClient(c)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range c {
			time.Sleep(5 * time.Millisecond)
		}
		server.RemoveClient(c)
	}()

	if err := server.StartBroadcast(sendData(), 5, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBroadcastServiceUnregisterChannel(t *testing.T) {
	server := initBroadcastService()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan string)
	server.AddClient(c)
	go func() {
		defer wg.Done()
		actual := string(<-c)
		assertEqual(t, testData[0], actual)
		server.RemoveClient(c)
	}()

	if err := server.StartBroadcast(sendData(), 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestBroadcastServiceDeadlock(t *testing.T) {
	server := initBroadcastService()
	var wg sync.WaitGroup
	wg.Add(2)

	c1 := make(chan string)
	server.AddClient(c1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		server.RemoveClient(c1)
	}()

	c2 := make(chan string)
	go func() {
		defer wg.Done()

		time.Sleep(time.Second)
		server.AddClient(c2)
		server.RemoveClient(c2)
	}()

	if err := server.StartBroadcast(sendData(), 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
