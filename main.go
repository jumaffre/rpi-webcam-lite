package main

import (
	"fmt"
	"net/http"
	"github.com/stianeikeland/go-rpio"
	"log"
)

func HelloServer(w http.ResponseWriter, r *http.Request) {
	
	err := rpio.Open()
	if err != nil {
		panic(fmt.Sprint("unable to open gpio", err.Error()))
	}
	defer rpio.Close()

	pin := rpio.Pin(26)

	pin.Output()
	pin.Toggle()
}

func main() {
	fmt.Println("Starting web server...")
	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServeTLS(":4443", "server.crt", "server.key", nil)
	if err != nil {
		log.Fatal("ListenAndServerTLS", err)
	}
}
