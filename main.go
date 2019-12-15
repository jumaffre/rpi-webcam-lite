package main

import (
	"fmt"
	"net/http"
	"github.com/stianeikeland/go-rpio"
	"log"
	auth "github.com/abbot/go-http-auth"
)

func HelloServer(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	
	fmt.Fprintf(w, "Toggling led")
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
	fmt.Println("Starting authenticated web server...")
	secrets := auth.HtdigestFileProvider("projecta.htdigest")
	authenticator := auth.NewDigestAuthenticator("/", secrets)

	http.HandleFunc("/", authenticator.Wrap(HelloServer))

	err := http.ListenAndServeTLS(":4443", "server.crt", "server.key", nil)
	if err != nil {
		log.Fatal("ListenAndServerTLS", err)
	}
}
