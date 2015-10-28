package main

import (
	"github.com/wblakecaldwell/pigeon"
	"log"
	"os"
)

func main() {
	worker, err := pigeon.NewWorker()
	if err != nil {
		log.Printf("Error creating new Worker: %s", err)
		os.Exit(1)
	}
	err = worker.ConnectToHub("localhost:5555")
	if err != nil {
		log.Printf("Error connecting to hub: %s", err)
		os.Exit(1)
	}
}
