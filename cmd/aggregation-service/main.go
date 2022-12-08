package main

import (
	"io"
	"log"
	"net/http"
)

func getAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "404 page not found")
		return
	}

	log.Println("Got Request")
	io.WriteString(w, "Placeholder.\n")
}

func main() {
	http.HandleFunc("/analysis", getAnalysis)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
