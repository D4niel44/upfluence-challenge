package sse

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

var testData = []string{
	"123",
	"4",
	"789abrhv",
	"100",
	"{data:1}",
	"event s",
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
			io.WriteString(w, fmt.Sprintf("event: 2\ndata: %s\n\n", line))
		}
	}))
}

func assertEqual(t *testing.T, expected string, actual string) {
	if actual != expected {
		t.Errorf("Expected %s, but got %s", expected, actual)
	}
}

func TestSSEClient(t *testing.T) {
	server := newTestServer()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan []byte)
	go func() {
		defer wg.Done()
		for _, expected := range testData {
			actual := string(<-c)
			assertEqual(t, expected, actual)
		}
	}()

	if err := StartListening(server.URL, c); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}
