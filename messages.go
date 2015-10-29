package pigeon

import (
	"fmt"
)

// ActionInfo can be performed by a worker
type ActionInfo struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	Description string `json:"description"`
}

// WorkerInfo describes a Worker's capabilities
type WorkerInfo struct {
	HostName         string       `json:"hostname"`
	AvailableActions []ActionInfo `json:"availableActions"`
}

// ActionRequest is a command sent from the Hub to the worker for execution
type ActionRequest struct {
	RequestID   int64  `json:"requestID"`
	HostName    string `json:"hostName"`
	CommandName string `json:"commandName"`
	Arguments   string `json:"arguments"`
}

// String returns a string representation of the ActionRequest
func (ar *ActionRequest) String() string {
	return fmt.Sprintf("RequestID: %d; HostName: %s; CommandName: %s; Arguments: %s;", ar.RequestID, ar.HostName, ar.CommandName, ar.Arguments)
}

// ActionResponse is a response sent from the Worker to the Hub in response to an ActionRequest
type ActionResponse struct {
	ActionRequest
	Response string `json:"response"`
	ErrorMsg string `json:"errorMsg"`
}
