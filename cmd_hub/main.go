package main

import (
	"github.com/wblakecaldwell/pigeon"
	"log"
	"net/http"
	"os"
)

func main() {
	serveMux := http.NewServeMux()
	hub, err := pigeon.NewHub("/connect", serveMux)
	if err != nil {
		log.Printf("Error creating a new Hub: %s", err)
		os.Exit(1)
	}
	go hub.Listen()
	http.ListenAndServe("localhost:5555", serveMux)
}
