package sse

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
)

const (
	dataPrefix = "data"
)

func StartListening(url string, out chan<- []byte) error {
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
			out <- bytes.TrimSpace(data[1])
		}

		if err == io.EOF {
			break
		}
	}
	return nil
}
