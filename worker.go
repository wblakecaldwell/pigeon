package pigeon

import (
	"fmt"
	"golang.org/x/net/websocket"
	"strings"
	"sync"
)

// Runner does something, given a request, returning a response
type Runner func(string) (string, error)

// Action is an action that this worker can perform
type Action interface {
	Name() string
	Run(string) (string, error)
}

// action is an Action implementation
type action struct {
	name   string
	runner Runner
}

// Name returns the name of this action
func (a *action) Name() string {
	return a.name
}

// Run performs the action, returning the response
func (a *action) Run(req string) (string, error) {
	return a.runner(req)
}

// NewAction returns a new Action
func NewAction(name string, runner Runner) Action {
	return &action{
		name:   name,
		runner: runner,
	}
}

// Worker declares different actions that it can run and report back
// to a host.
type Worker struct {
	actionByName map[string]Action // keep track of actions by name
	actionsLock  sync.RWMutex      // lock for accessing the actions maps
	ws           *websocket.Conn   // connection to the Hub
	doneCh       chan interface{}  // closed when done
}

// NewWorker returns a new Worker
func NewWorker() (*Worker, error) {
	return &Worker{
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

	ws.Write([]byte("Hello!"))

	return nil
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
