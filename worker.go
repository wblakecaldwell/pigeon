package pigeon

//
// Worker
//

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"strings"
	"sync"
	"time"
)

// Runner does something, given a request, returning a response
type Runner func(string) (string, error)

// Worker declares different actions that it can run and report back
// to a host.
type Worker struct {
	host           string            // host of this worker
	actionByName   map[string]Action // keep track of actions by name
	actionsLock    sync.RWMutex      // lock for accessing the actions maps
	doneCh         chan interface{}  // closed when done
	inMessageChan  chan Message      // channel populated by messages received from the Hub
	outMessageChan chan Message      // channel we write outgoing messages to, and deliver to the Hub
}

// NewWorker returns a new Worker
func NewWorker(host string) (*Worker, error) {
	return &Worker{
		host:           host,
		actionByName:   make(map[string]Action),
		actionsLock:    sync.RWMutex{},
		inMessageChan:  make(chan Message, 100),
		outMessageChan: make(chan Message, 100),
	}, nil
}

// ConnectToHub maintains a registered connection to the input hub host and port
// TODO: pass errors back to caller - have caller pass in an error channel? Return one?
func (w *Worker) ConnectToHub(hostPort string) error {
	var ws *websocket.Conn // connection to the Hub
	wsMutex := sync.RWMutex{}

	// ensure connected - if there's any problem, sets the websocket to nil, and returns error
	ensureConnected := func() error {
		var err error
		url := fmt.Sprintf("ws://%s/connect", hostPort)

		if ws == nil {
			fmt.Printf("Dialing the Hub at %s\n", url)
			ws, err = websocket.Dial(url, "", "http://localhost") // TODO: protocol needed?
			if err != nil {
				ws = nil
				fmt.Printf("Failure dialing the Hub at %s: %s\n", url, err)
				return fmt.Errorf("Error dialing Hub: %s", err)
			}
			fmt.Printf("Success dialing the Hub at %s\n", url)

			// send the worker info
			workerInfo, err := w.WorkerInfo()
			if err != nil {
				ws = nil
				return fmt.Errorf("Error getting the WorkerInfo - closing the connection: %s\n", err)
			}
			err = websocket.JSON.Send(ws, workerInfo)
			if err != nil {
				if closeErr := ws.Close(); closeErr != nil {
					fmt.Printf("Error closing websocket: %s\n", closeErr)
					// don't return
				}
				ws = nil
				return fmt.Errorf("Error sending the WorkerInfo - closing the connection: %s", err)
			}
			fmt.Println("Hub connection re-established")
			return nil
		}

		// send a test message
		fmt.Printf("Sending a test message to the Hub to check connectivity at %s\n", url)
		message := HelloMessage{}
		err = websocket.JSON.Send(ws, message)
		if err != nil {
			ws = nil
			return fmt.Errorf("Failure sending test message to Hub: %s", err)
		}
		return nil
	}

	getWsFunc := func() (*websocket.Conn, error) {
		for {
			wsMutex.RLock()
			localWs := ws
			wsMutex.RUnlock()

			if localWs != nil {
				return localWs, nil
			}

			time.Sleep(6 * time.Second)

			// TODO: way to break out
		}

		return nil, fmt.Errorf("Couldn't get the websocket connection")
	}

	// maintain the connection
	go func() {
		for {
			wsMutex.Lock()
			ensureConnected()
			wsMutex.Unlock()

			time.Sleep(5 * time.Second)

			// TODO: add way to break out
		}
	}()

	// listen for incoming messages
	go func() {
		// loop: read each message
		for {
			localWs, err := getWsFunc()
			if err != nil {
				fmt.Printf("Error trying to get a webservice connection: %s\n", err)
				continue
			}

			// fetch the next message
			var message Message
			err = websocket.JSON.Receive(localWs, &message)
			if err != nil {
				fmt.Printf("Error trying to receive messages from websocket: %s\n", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			w.inMessageChan <- message

			// TODO: add way to break out
		}
	}()

	// listen for outgoing messages
	go func() {
		for {
			select {
			case message := <-w.outMessageChan:
				localWs, err := getWsFunc()
				if err != nil {
					fmt.Printf("Error trying to get a webservice connection: %s\n", err)
					continue
				}

				err = websocket.JSON.Send(localWs, message)
				if err != nil {
					fmt.Printf("Error received while trying to send message %#v - queuing it back up: %s\n", message, err)
					w.outMessageChan <- message
					continue
				}

			case <-w.doneCh:
				fmt.Println("Outgoing message loop: doneChan is closed - breaking out")
				return
			}
		}
	}()

	// main loop - handle incoming requests
	for {
		select {
		case message := <-w.inMessageChan:
			fmt.Printf("Received message from Hub: %#v\n", message) // TODO: this is a low-level debug message

			switch message.Type {
			case "action-request":
				fmt.Println("Received message is an ActionRequestMessage")

				var actionRequest ActionRequestMessage
				err := json.Unmarshal(*message.Message, &actionRequest)
				if err != nil {
					fmt.Printf("Couldn't unmarshal ActionRequestMessage: %s\n", err)
					continue
				}

				// TODO: do this in a goroutine
				// run the action
				actionResponse, err := w.ExecuteAction(actionRequest)
				if err != nil {
					fmt.Printf("Error executing the action-request %#v: %s\n", actionRequest, err)
					continue
				}
				fmt.Printf("Executed response for %#v: %#v\n", actionRequest, actionResponse)

				// queue up the response
				respJSON, err := json.Marshal(actionResponse)
				if err != nil {
					fmt.Println("Error JSON-marshalling the action-response response. Request: %#v; Response: %#v: %s",
						actionRequest, actionResponse, err)
					continue
				}

				rawMessage := json.RawMessage(respJSON)
				w.outMessageChan <- Message{
					Type:    "action-response",
					Message: &rawMessage,
				}

				fmt.Printf("Queued action-request response: %#v\n", actionResponse)
			}
		}
	}

	return nil
}

// ExecuteAction runs the input ActionRequest
// - any error executing the command that should be send back to the caller should
//   be written to the response, and nil message returned from this method.
func (w *Worker) ExecuteAction(actionRequest ActionRequestMessage) (*ActionResponseMessage, error) {
	w.actionsLock.RLock()
	defer w.actionsLock.RUnlock()

	response := &ActionResponseMessage{
		ActionRequestMessage: actionRequest,
	}

	// find the action
	action, ok := w.actionByName[strings.ToLower(actionRequest.CommandName)]
	if !ok {
		response.ErrorMsg = fmt.Sprintf("Could not find an action named %s", actionRequest.CommandName)
		return response, fmt.Errorf("Could not find an action registered with the name %s", actionRequest.CommandName)
	}

	// perform the action
	responseText, err := action.Run(actionRequest.Arguments)
	if err != nil {
		response.ErrorMsg = fmt.Sprintf("An error occurred while running the command")
		return response, fmt.Errorf("An error occurred while performing the action: %#v - %s", actionRequest, err)
	}
	response.Response = responseText

	return response, nil
}

// Disconnect stops the connection to the remote Hub
func (w *Worker) Disconnect() {
	//TODO
}

// RegisterAction adds an action to the list
func (w *Worker) RegisterAction(action Action) error {
	// check if exists already
	w.actionsLock.RLock()
	name := strings.ToLower(action.Name())
	if _, ok := w.actionByName[name]; ok {
		w.actionsLock.RUnlock()
		return fmt.Errorf("There is already an action named %s", name)
	}
	w.actionsLock.RUnlock()

	// insert into the map
	w.actionsLock.Lock()
	defer w.actionsLock.Unlock()
	w.actionByName[name] = action
	return nil
}

// GetAction gets an action by name
func (w *Worker) GetAction(name string) Action {
	w.actionsLock.RLock()
	defer w.actionsLock.RUnlock()

	name = strings.ToLower(name)
	if foundAction, ok := w.actionByName[name]; ok {
		return foundAction
	}
	return nil
}

// WorkerInfo builds a WorkerInfo about this Worker and its Actions
func (w *Worker) WorkerInfo() (WorkerInfo, error) {
	info := WorkerInfo{
		HostName: w.host,
	}

	w.actionsLock.Lock()
	defer w.actionsLock.Unlock()
	for _, registeredAction := range w.actionByName {
		info.AvailableActions = append(info.AvailableActions, ActionInfo{
			Name:        registeredAction.Name(),
			Usage:       registeredAction.Usage(),
			Description: registeredAction.Description(),
		})
	}

	return info, nil
}

// NewActionInfo returns an ActionInfo from an Action interface
func NewActionInfo(action Action) (ActionInfo, error) {
	return ActionInfo{
		Name:        action.Name(),
		Usage:       action.Usage(),
		Description: action.Description(),
	}, nil
}
