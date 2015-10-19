package host

import (
	"fmt"
	"strings"
	"sync"
)

// Runner returns a byte array
type Runner func() ([]byte, error)

// Action is an action that this host can perform
type Action interface {
	Name() string
	Run() ([]byte, error)
}

type action struct {
	name   string
	runner Runner
}

func (a *action) Name() string {
	return a.name
}
func (a *action) Run() ([]byte, error) {
	return a.runner()
}

// NewAction returns a new Action
func NewAction(name string, runner Runner) Action {
	return &action{
		name:   name,
		runner: runner,
	}
}

// Host declares different actions that it can run and report back
// to a receiver.
type Host struct {
	actionByName map[string]Action // keep track of actions by name
	actionsLock  sync.RWMutex      // lock for accessing the actions maps
}

// NewHost returns a new Host
func NewHost() (*Host, error) {
	return &Host{
		actionByName: make(map[string]Action),
		actionsLock:  sync.RWMutex{},
	}, nil
}

// RegisterAction adds an action to the list
func (s *Host) RegisterAction(action Action) error {
	// check if exists already
	s.actionsLock.RLock()
	name := strings.ToLower(action.Name())
	if _, ok := s.actionByName[name]; ok {
		s.actionsLock.RUnlock()
		return fmt.Errorf("There is already an action named %s", name)
	}
	s.actionsLock.RUnlock()

	// insert into the map
	s.actionsLock.Lock()
	defer s.actionsLock.Unlock()
	s.actionByName[name] = action
	return nil
}

// GetAction gets an action by name
func (s *Host) GetAction(name string) Action {
	s.actionsLock.RLock()
	defer s.actionsLock.RUnlock()

	name = strings.ToLower(name)
	if foundAction, ok := s.actionByName[name]; ok {
		return foundAction
	}
	return nil
}
