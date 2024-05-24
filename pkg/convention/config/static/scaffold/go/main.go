package main

import (
	"encoding/json"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	headers, _ := json.Marshal(r.Header)
	w.Header().Set("Content-Type", "application/json")
	w.Write(headers)
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
