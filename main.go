package main

import (
    "log"
    "net/http"
)

func serveIndex(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.Error(w, "not found", http.StatusNotFound)
    }

    if r.Method != "GET" {
        http.Error(w, "not found", http.StatusNotFound)
    }
    http.ServeFile(w, r, "templates/index.html")
}

func main() {
    http.HandleFunc("/", serveIndex)
    // http.HandleFunc("/ws", nil)

    log.Fatal(http.ListenAndServe(":3000", nil))
}
