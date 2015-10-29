package pigeon

import (
	"fmt"
	"golang.org/x/net/websocket"
	"strings"
	"sync"
)

// Runner does something, given a request, returning a response
type Runner func(string) (string, error)

// Worker declares different actions that it can run and report back
// to a host.
type Worker struct {
	host         string            // host of this worker
	actionByName map[string]Action // keep track of actions by name
	actionsLock  sync.RWMutex      // lock for accessing the actions maps
	ws           *websocket.Conn   // connection to the Hub
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

// ConnectToHub maintains a registered connection to the input hub host and port
// TODO: pass errors back to caller - have caller pass in an error channel? Return one?
func (w *Worker) ConnectToHub(hostPort string) error {
	origin := "http://localhost"
	url := fmt.Sprintf("ws://%s/connect", hostPort)

	// TODO: keep the connection alive, and set up channels for send/receive
	ws, err := websocket.Dial(url, "", origin) // TODO: protocol needed?
	if err != nil {
		return fmt.Errorf("Error dialing to Hub: %s", err)
	}

	w.ws = ws
	workerInfo, _ := w.WorkerInfo() // TODO: error
	err = websocket.JSON.Send(ws, workerInfo)
	if err != nil {
		fmt.Printf("Error sending WorkerInfo to the hub")
	}
	fmt.Println("Sent message to Hub - now listening for requests from Hub")

	// listen for action requests from Hub
	for {
		actionRequest := ActionRequest{}
		err := websocket.JSON.Receive(w.ws, &actionRequest)
		if err != nil {
			fmt.Printf("Error receiving ActionRequest from Hub: %s", err)
			continue
		}

		actionResponse, err := w.ExecuteAction(actionRequest)
		if err != nil {
			fmt.Printf("Error occurred while trying to execute a request: %s", err)
			continue
		}
		fmt.Println("Sending response")
		err = websocket.JSON.Send(ws, actionResponse)
		if err != nil {
			fmt.Printf("Error occurred while trying to submit response: %s", err)
			continue
		}
	}

	return nil
}

// ExecuteAction runs the input ActionRequest
func (w *Worker) ExecuteAction(actionRequest ActionRequest) (*ActionResponse, error) {
	w.actionsLock.RLock()
	defer w.actionsLock.RUnlock()

	response := &ActionResponse{
		ActionRequest: actionRequest,
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
		return response, fmt.Errorf("An error occurred while performing the action: %s - %s", actionRequest.String(), err)
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
