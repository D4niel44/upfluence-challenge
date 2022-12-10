package sse

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, line := range testData {
			io.WriteString(w, fmt.Sprintf("data: %s\n\n", line))
		}
	}))
}

func assertEqual(t *testing.T, expected string, actual string) {
	if actual != expected {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}

func TestNewSSEClient(t *testing.T) {
	f := func(b []byte) ([]byte, error) { return b, nil }
	client := NewSSEClient(f)
	if client.channels == nil {
		t.Fatal("Map was not initialized")
	}
	if client.parseFunc == nil {
		t.Fatal("Parse function was not initialized")
	}
}

func TestSSEClientOneChannel(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan []byte)
	RegisterChannel(c)
	defer UnregisterChannel(c)
	go func() {
		defer wg.Done()
		for _, expected := range testData {
			actual := string(<-c)
			assertEqual(t, expected, actual)
		}
	}()

	if err := StartListening(server.URL, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestSSEClientMultipleChannels(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < len(testData); i++ {
		c := make(chan []byte)
		RegisterChannel(c)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < i; j++ {
				actual := string(<-c)
				assertEqual(t, testData[j], actual)
			}
			UnregisterChannel(c)
		}(i)
	}
	if err := StartListening(server.URL, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestSSEClientMultipleWorkers(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < len(testData); i++ {
		c := make(chan []byte)
		RegisterChannel(c)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < i; j++ {
				<-c
			}
			UnregisterChannel(c)
		}(i)
	}
	if err := StartListening(server.URL, 10, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestSSEClientTimeout(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(len(testData))
	for i := 0; i < 10; i++ {
		c := make(chan []byte)
		RegisterChannel(c)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Second)
			UnregisterChannel(c)
		}()
	}

	c := make(chan []byte)
	RegisterChannel(c)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range c {
			time.Sleep(5 * time.Millisecond)
		}
		UnregisterChannel(c)
	}()

	if err := StartListening(server.URL, 5, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestSSEClientUnregisterChannel(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan []byte)
	RegisterChannel(c)
	go func() {
		defer wg.Done()
		actual := string(<-c)
		assertEqual(t, testData[0], actual)
		UnregisterChannel(c)
	}()

	if err := StartListening(server.URL, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestSSEClientDeadlock(t *testing.T) {
	server := newTestServer()
	var wg sync.WaitGroup
	wg.Add(2)

	c1 := make(chan []byte)
	RegisterChannel(c1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		UnregisterChannel(c1)
	}()

	c2 := make(chan []byte)
	go func() {
		defer wg.Done()

		time.Sleep(time.Second)
		RegisterChannel(c2)
		UnregisterChannel(c2)
	}()

	if err := StartListening(server.URL, 1, time.Second); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
