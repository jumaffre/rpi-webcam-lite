package main

import (
	"fmt"
	"net/http"
)

func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s", r.URL)
}

func main() {
	fmt.Println("Starting web server...")
	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8080", nil)
}
