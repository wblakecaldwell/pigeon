package main

import (
	"github.com/wblakecaldwell/pigeon"
	"log"
	"os"
)

func main() {
	worker, err := pigeon.NewWorker("localhost")
	if err != nil {
		log.Printf("Error creating new Worker: %s", err)
		os.Exit(1)
	}

	// info about the worker
	runner := func(args string) (string, error) {
		return "Hello, " + args, nil
	}
	action := pigeon.NewAction("say-hello", "say-hello <some name>", "Says hello to whomever you want!", runner)
	err = worker.RegisterAction(action)
	if err != nil {
		log.Printf("Error registering action '%s': %s", "say-hello", err)
		os.Exit(1)
	}

	err = worker.Start("localhost:5555")
	if err != nil {
		log.Printf("Error connecting to hub: %s", err)
		os.Exit(1)
	}
}
