package pigeon

//
// Worker
//

import (
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
	host         string            // host of this worker
	actionByName map[string]Action // keep track of actions by name
	actionsLock  sync.RWMutex      // lock for accessing the actions maps
	doneCh       chan interface{}  // closed when done
}

// NewWorker returns a new Worker
func NewWorker(host string) (*Worker, error) {
	return &Worker{
		host:         host,
		actionByName: make(map[string]Action),
		actionsLock:  sync.RWMutex{},
	}, nil
}

// Start maintains a registered connection to the input hub host and port
// TODO: pass errors back to caller - have caller pass in an error channel? Return one?
func (w *Worker) Start(hostPort string) error {

	// function that dials and registers capabilities if needed
	repairFunc := func() (*websocket.Conn, error) {
		fmt.Println("Need to repair connection")

		url := fmt.Sprintf("ws://%s/connect", hostPort)
		for {
			fmt.Printf("Dialing the Hub at %s\n", url)
			ws, err := websocket.Dial(url, "", "http://localhost") // TODO: protocol needed?
			if err != nil {
				ws = nil
				fmt.Printf("Failure dialing the Hub at %s: %s\n", url, err)
				time.Sleep(500 * time.Millisecond)
				continue
			}
			fmt.Printf("Success dialing the Hub at %s\n", url)

			// send the worker info
			message, err := NewMessage("worker-info", w.WorkerInfo())
			if err != nil {
				// this is fatal - can't recover from this nonsense
				return nil, fmt.Errorf("Can't create the WorkerInfo Message: %s", err)
			}

			err = websocket.JSON.Send(ws, message)
			if err != nil {
				fmt.Printf("Error sending the WorkerInfo Message: %s\n", err)
				continue
			}
			fmt.Println("Hub connection re-established")
			return ws, nil
		}
	}

	wsClient := newWebsocketClient(nil, repairFunc)
	wsClient.Start()

	// main loop - handle incoming requests
	for {
		// TODO: quit channel, which stops the wsClient and breaks out
		select {
		case message := <-wsClient.InMessageChan:
			prettyLogMessage("Received message from Hub:", message)

			switch message.Type {
			case "action-request":
				fmt.Println("Handling an ActionRequest (type \"action-request\")")
				go w.handleActionRequest(wsClient, &message)
			}
		}
	}

	return nil
}

// handleActionRequest handles a message of type "action-request" - is run in a goroutine
func (w *Worker) handleActionRequest(wsClient *websocketClient, message *Message) {
	actionRequest, err := ExtractActionRequestMessage(message)
	if err != nil {
		fmt.Printf("Failure extracting message - type is \"action-request\": %s\n", err)
		return
	}

	// run the action
	actionResponse, err := w.ExecuteAction(*actionRequest)
	if err != nil {
		fmt.Printf("Error executing the action-request %#v: %s\n", actionRequest, err)
		return
	}
	prettyLogMessage("Executed response - request/response:", actionRequest, actionResponse)

	// queue up the response
	responseMessage, err := NewMessage("action-response", actionResponse)
	if err != nil {
		fmt.Println("Error creating Message object: %s", err)
		return
	}
	wsClient.OutMessageChan <- *responseMessage
	prettyLogMessage("Queued action-request response:", actionResponse)
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
func (w *Worker) WorkerInfo() WorkerInfo {
	info := WorkerInfo{
		HostName: w.host,
	}

	w.actionsLock.Lock()
	defer w.actionsLock.Unlock()
	for _, registeredAction := range w.actionByName {
		info.AvailableActions = append(info.AvailableActions, NewActionInfo(registeredAction))
	}

	return info
}

// NewActionInfo returns an ActionInfo from an Action interface
func NewActionInfo(action Action) ActionInfo {
	return ActionInfo{
		Name:        action.Name(),
		Usage:       action.Usage(),
		Description: action.Description(),
	}
}
