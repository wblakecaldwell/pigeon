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
	quitChan               chan interface{}
}

// newWebsocketClient returns a new websocketClient
func newWebsocketClient(conn *websocket.Conn, repairConnectivityFunc connectionRepairFunc) *websocketClient {
	if repairConnectivityFunc == nil {
		repairConnectivityFunc = func() (*websocket.Conn, error) {
			return nil, fmt.Errorf("Can't/won't re-establish connection")
		}
	}

	return &websocketClient{
		wsConn:                 conn,
		wsConnMutex:            sync.RWMutex{},
		InMessageChan:          make(chan Message, 1000),
		OutMessageChan:         make(chan Message, 1000),
		repairConnectivityFunc: repairConnectivityFunc,
		quitChan:               make(chan interface{}),
	}
}

// Start starts the connection maintenance, read, and write goroutines. Does not block.
func (client *websocketClient) Start() {
	go client.maintainConnectivity()
	go client.receiveMessages()
	go client.sendMessages()
}

// Stop shuts everything down
func (client *websocketClient) Stop() {
	close(client.quitChan)
}

// MaintainConnectivity sends a HelloMessage to check if connected. If not connected,
// will try to repair the connection.
func (client *websocketClient) maintainConnectivity() {
	defer close(client.quitChan)

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
func (client *websocketClient) receiveMessages() {
	for {
		select {
		case _ = <-client.quitChan:
			// told to stop, or disconnected and couldn't reconnect
			fmt.Println("quitting: receiveMessages")
			return
		default:
		}

		client.wsConnMutex.RLock()
		ws := client.wsConn
		client.wsConnMutex.RUnlock()

		if ws == nil {
			// not currently connected - try again in 500ms
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

		// accept, but outwardly ignore "hello" messages
		if message.Type == "hello" {
			fmt.Println("Received hello message")
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
		case <-client.quitChan:
			fmt.Println("quitting: sendMessages")
			return
		}
	}
}
