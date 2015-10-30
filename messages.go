package pigeon

import (
	"encoding/json"
	"fmt"
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

// NewMessage returns a Message with the input type and populated sub-mssage
func NewMessage(messageType string, message interface{}) (*Message, error) {
	subMsgJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("Error JSON marshalling the struct type %s: %#v", messageType, message)
	}
	rawSubMsg := json.RawMessage(subMsgJSON)
	return &Message{
		Type:    messageType,
		Message: &rawSubMsg,
	}, nil
}

// ActionRequestMessage is a top-level Message type 'action-request' command sent from the Hub to the worker for execution
type ActionRequestMessage struct {
	RequestID   int64  `json:"requestID"`
	HostName    string `json:"hostName"`
	CommandName string `json:"commandName"`
	Arguments   string `json:"arguments"`
}

// ExtractActionRequestMessage unmarshals an ActionRequestMessage from the input Message's Message property,
// returning error if it's the wrong type
func ExtractActionRequestMessage(message *Message) (*ActionRequestMessage, error) {
	ret := &ActionRequestMessage{}
	err := extractMessage(message, "action-request", ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ActionResponseMessage is a top-level Message type 'action-response' response sent from the Worker to the Hub in response to an ActionRequest
type ActionResponseMessage struct {
	ActionRequestMessage
	Response string `json:"response"`
	ErrorMsg string `json:"errorMsg"`
}

// ExtractActionResponseMessage unmarshals an ActionResponseMessage from the input Message's Message property,
// returning error if it's the wrong type
func ExtractActionResponseMessage(message *Message) (*ActionResponseMessage, error) {
	ret := &ActionResponseMessage{}
	err := extractMessage(message, "action-response", ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// extractMessage tries to unmarshal a Message.Message
func extractMessage(message *Message, messageType string, target interface{}) error {
	if message.Type != messageType {
		return fmt.Errorf("The message is of type \"%s\", not \"%s\"", message.Type, messageType)
	}
	err := json.Unmarshal(*message.Message, &target)
	if err != nil {
		return fmt.Errorf("Error received while unmarshalling message type \"%s\": %#v - %s",
			messageType, string(*message.Message), err)
	}
	return nil
}
