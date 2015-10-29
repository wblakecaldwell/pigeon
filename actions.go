package pigeon

// Action is an action that this worker can perform
type Action interface {
	Name() string
	Usage() string
	Description() string
	Run(string) (string, error)
}

// action is an Action implementation
type action struct {
	name        string
	usage       string
	description string
	runner      Runner
}

// Name returns the name of this action
func (a *action) Name() string {
	return a.name
}

// Usage describes how to use the action
func (a *action) Usage() string {
	return a.usage
}

// Description returns a description of this action
func (a *action) Description() string {
	return a.description
}

// Run performs the action, returning the response
func (a *action) Run(req string) (string, error) {
	return a.runner(req)
}

// NewAction returns a new Action
func NewAction(name string, usage string, description string, runner Runner) Action {
	return &action{
		name:        name,
		usage:       usage,
		description: description,
		runner:      runner,
	}
}
