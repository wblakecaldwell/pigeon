package pigeon

import (
	"encoding/json"
)

// Message is the container for all websocket messages
type Message struct {
	Type    string           `json:"type"`
	Message *json.RawMessage `json:"message"`
}

// ActionInfo can be performed by a worker
type ActionInfo struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	Description string `json:"description"`
}

// WorkerInfo is a top-level Message 'worker-info' type that describes a Worker's capabilities
type WorkerInfo struct {
	HostName         string       `json:"hostname"`
	AvailableActions []ActionInfo `json:"availableActions"`
}

// HelloMessage is a top-level Message 'hello' that's sent, and ignored, to check connection.
type HelloMessage struct {
	// ignored
}

// ActionRequestMessage is a top-level Message type 'action-request' command sent from the Hub to the worker for execution
type ActionRequestMessage struct {
	RequestID   int64  `json:"requestID"`
	HostName    string `json:"hostName"`
	CommandName string `json:"commandName"`
	Arguments   string `json:"arguments"`
}

// ActionResponseMessage is a top-level Message type 'action-response' response sent from the Worker to the Hub in response to an ActionRequest
type ActionResponseMessage struct {
	ActionRequestMessage
	Response string `json:"response"`
	ErrorMsg string `json:"errorMsg"`
}
