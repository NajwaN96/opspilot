package main

import (
  "log"
  "net/http"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
    w.Write([]byte("ok"))
  })
  mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) {
    w.Write([]byte("ready"))
  })
  mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
    w.Write([]byte("# HELP http_requests_total synthetic\nhttp_requests_total 0\n"))
  })
  log.Fatal(http.ListenAndServe(":8080", mux))
}
