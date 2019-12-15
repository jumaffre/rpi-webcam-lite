package main

import (
	"fmt"
	"net/http"
	"github.com/stianeikeland/go-rpio"
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
	http.ListenAndServe(":8080", nil)
}
