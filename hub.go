// Package pigeon describes a node that Workers connect to, waiting
// to receive requests from.
package pigeon

import (
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

		wsClient := newWebsocketClient(ws, nil)
		wsClient.Start()
		defer wsClient.Stop()

		go func() {
			// TODO - Temporary: send a test action request
			actionRequestMessage := ActionRequestMessage{
				RequestID:   123,
				HostName:    "localhost",
				CommandName: "say-hello",
				Arguments:   "Blake",
			}
			message, err := NewMessage("action-request", actionRequestMessage)
			if err != nil {
				// can't recover from this
				fmt.Printf("Error creating action-request: %s\n", err)
				return
			}
			wsClient.OutMessageChan <- *message
		}()

		var workerInfo *WorkerInfo
		var err error

		// main loop
		for {
			// TODO: quit channel, which stops the wsClient and breaks out
			select {
			case message := <-wsClient.InMessageChan:
				switch message.Type {
				case "worker-info":
					// register/update worker info
					workerInfo, err = ExtractWorkerInfoMessage(&message)
					if err != nil {
						// dont' recover from this
						fmt.Printf("Error extracting WorkerInfo message: %s\n", err)
						return
					}
					// TODO: re-register Worker capabilities
					prettyLogMessage("Received worker capabilities:", &workerInfo)
					continue

				case "action-response":
					actionResponse, err := ExtractActionResponseMessage(&message)
					if err != nil {
						fmt.Printf("Error extracting ActionResponseMessage: %s\n", err)
						continue
					}

					// TODO: handle the response
					prettyLogMessage("Response received frmo say-hello action:", &actionResponse)
				}
			}
		}
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
