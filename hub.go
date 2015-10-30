// Package pigeon describes a node that Workers connect to, waiting
// to receive requests from.
package pigeon

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
)

// Hub is a websocket host, receiving connections from Workers
// see: https://github.com/golang-samples/websocket/blob/master/websocket-chat/src/chat/server.go
type Hub struct {
	urlPattern string
	serveMux   *http.ServeMux
	workers    map[int]*remoteWorker
	addCh      chan *remoteWorker
	delCh      chan *remoteWorker
	doneCh     chan bool
	errCh      chan error
}

// NewHub returns a new hub that listens on the input pattern
func NewHub(urlPattern string, serveMux *http.ServeMux) (*Hub, error) {
	return &Hub{
		urlPattern: urlPattern,
		serveMux:   serveMux,
	}, nil
}

func (h *Hub) add(worker *remoteWorker) {
	h.addCh <- worker
}

// Done stops the server from listening
func (h *Hub) Done() {
	close(h.doneCh)
}

// Listen for incoming requests from workers
func (h *Hub) Listen() {
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				h.errCh <- err
			}
		}()
		log.Println("Received a connection request")

		// TODO: ask the worker about itself, to populate remoteWorker{}
		// TODO: ask the remote worker what actions can be performed
		//worker := remoteWorker{}

		workerInfo := WorkerInfo{}
		err := websocket.JSON.Receive(ws, &workerInfo)
		if err != nil {
			log.Printf("Couldn't establish connection with remote worker: %s", err)
			return
		}
		fmt.Printf("TODO: Received worker message: %#v\n", workerInfo)
		//h.add(&worker)

		// WBCTODO: Test message
		// service this runner
		actionRequestMessage := ActionRequestMessage{
			RequestID:   123,
			HostName:    "localhost",
			CommandName: "say-hello",
			Arguments:   "Blake",
		}
		messageBody, err := json.Marshal(actionRequestMessage)
		if err != nil {
			fmt.Printf("Failure JSON marshalling: %s\n", err)
			panic("WBCTODO")
		}
		fmt.Printf("WBCTODO: %s\n", messageBody)

		rawMessage := json.RawMessage(messageBody)
		message := Message{
			Type:    "action-request",
			Message: &rawMessage,
		}
		err = websocket.JSON.Send(ws, message)
		if err != nil {
			fmt.Printf("Error received while sending 'say-hello' action request: %s", err)
			return
		}

		response := &Message{}
		err = websocket.JSON.Receive(ws, response)
		if err != nil {
			fmt.Println("Error received while trying to receive response for 'say-hello' action request: %s", err)
			return
		}
		actionResponse, err := ExtractActionResponseMessage(response)
		if err != nil {
			fmt.Println("Error receiving response: %s", err)
			return
		}
		fmt.Printf("\n\nResponse received from say-hello action!: %#v\n\n", actionResponse)
	}

	h.serveMux.Handle(h.urlPattern, websocket.Handler(onConnected))
	log.Printf("Listening on %s", h.urlPattern)

	for {
		select {
		case rw := <-h.addCh:
			log.Printf("Added a new RemoteWorker: %s", rw.String())
			// TODO: register the RemoteWorker

		case <-h.doneCh:
			log.Printf("Done listening on %s\n", h.urlPattern)
			return

		}
	}
}
