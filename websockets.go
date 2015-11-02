package pigeon

import (
	"fmt"
	"golang.org/x/net/websocket"
	"sync"
	"time"
)

type connectionRepairFunc func() (*websocket.Conn, error)

// websocketClient is an internal websocket manager for a websocket client
type websocketClient struct {
	wsConn                 *websocket.Conn // websocket connection
	wsConnMutex            sync.RWMutex
	InMessageChan          chan Message
	OutMessageChan         chan Message
	repairConnectivityFunc connectionRepairFunc // function we'll call to repair connectivitiy
}

// newWebsocketClient returns a new websocketClient
func newWebsocketClient(repairConnectivityFunc connectionRepairFunc) *websocketClient {
	return &websocketClient{
		wsConn:                 nil,
		wsConnMutex:            sync.RWMutex{},
		InMessageChan:          make(chan Message, 1000),
		OutMessageChan:         make(chan Message, 1000),
		repairConnectivityFunc: repairConnectivityFunc,
	}
}

// Start starts the connection maintenance, read, and write goroutines. Does not block.
func (client *websocketClient) Start() {
	go client.maintainConnectivity()
	go client.receiveMessages()
	go client.sendMessages()
}

// MaintainConnectivity sends a HelloMessage to check if connected. If not connected,
// will try to repair the connection.
func (client *websocketClient) maintainConnectivity() {
	for {
		client.wsConnMutex.RLock()
		ws := client.wsConn
		client.wsConnMutex.RUnlock()

		needToRepair := false
		if ws == nil {
			needToRepair = true
		} else {
			fmt.Println("Sending a test message to check connectivity")
			message, err := NewMessage("hello", HelloMessage{})
			if err != nil {
				fmt.Printf("Error creating Message from HelloMessage: %s\n", err)
			} else {
				err = websocket.JSON.Send(ws, message)
				if err != nil {
					fmt.Printf("Failure sending test message: %s\n", err)
					needToRepair = true
				}
			}
		}

		if needToRepair == true {
			success := func() bool {
				client.wsConnMutex.Lock()
				defer client.wsConnMutex.Unlock()

				newWs, err := client.repairConnectivityFunc()
				if err != nil {
					fmt.Printf("Websocket connection is dead - can't repair it: %s\n", err)
					return false
				}
				client.wsConn = newWs
				return true
			}()

			if !success {
				return
			}
		} else {
			fmt.Println("Success: test message sent")
		}

		time.Sleep(5 * time.Second)
	}
}

// receiveMessages keeps trying to read from a websocket connection,
// writing received messages to a channel
func (client *websocketClient) receiveMessages() error {
	for {
		client.wsConnMutex.RLock()
		ws := client.wsConn
		client.wsConnMutex.RUnlock()

		if ws == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// fetch the next message
		var message Message
		err := websocket.JSON.Receive(ws, &message)
		if err != nil {
			fmt.Printf("Error trying to receive messages from websocket: %s\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		client.InMessageChan <- message
	}
}

// sendMessages keeps trying to send any messages in the OutMessageChan
func (client *websocketClient) sendMessages() {
	for {
		select {
		case message := <-client.OutMessageChan:
			client.wsConnMutex.RLock()
			ws := client.wsConn
			client.wsConnMutex.RUnlock()

			if ws != nil {
				fmt.Printf("Sending message %#v\n", message)
				err := websocket.JSON.Send(ws, message)
				if err == nil {
					continue
				}
				fmt.Printf("Error trying to send message: %s\n", err)
			} else {
				fmt.Println("Error trying to send message: websocket connection is nil")
			}

			client.OutMessageChan <- message
			time.Sleep(500 * time.Millisecond)
			// TODO: quit channel
		}
	}
}
