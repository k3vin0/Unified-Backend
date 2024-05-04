package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust the origin checking to ensure proper security
	},
}
var clients = make(map[*websocket.Conn]string) // Map connections to user identifiers

var mutex sync.Mutex // to protect the clients map

type Message struct {
	UserId    string `json:"userId"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type MessageShape struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

var messageHistory []Message // Store all messages sent during the session

// WebSocketEchoHandler handles WebSocket upgrade requests
// Broadcast channel
var broadcast = make(chan Message)

// HandleWebSocketConnection handles WebSocket upgrade requests and manages messaging
func HandleWebSocketConnection(c echo.Context) error {
	userId := c.QueryParam("userId") // Example: Extracting userID from query parameter
	if userId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "UserID is required")
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// Register new client with user identifier
	mutex.Lock()
	clients[ws] = userId
	fmt.Print(messageHistory)
	for _, msg := range messageHistory {
		jsonData, err := json.Marshal(msg)
		if err == nil { // Only send if the JSON marshaling succeeds
			ws.WriteMessage(websocket.TextMessage, jsonData)
		}
	}
	mutex.Unlock()

	defer func() {
		// Remove client on disconnect
		mutex.Lock()
		delete(clients, ws)
		if len(clients) == 0 {
			fmt.Println("No more clients connected, deleting chat history...")
			messageHistory = []Message{}
		}
		mutex.Unlock()
	}()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error("read:", err)
			break
		}
		var messageObject MessageShape
		errorMessage := json.Unmarshal([]byte(message), &messageObject)
		if errorMessage != nil {
			return errorMessage
		}
		newMessage := Message{UserId: userId, Message: string(messageObject.Message), Timestamp: messageObject.Timestamp}
		mutex.Lock()
		messageHistory = append(messageHistory, newMessage) // Save message to history
		mutex.Unlock()
		// Broadcast message along with userId
		broadcast <- newMessage
	}
	return nil
}

// StartBroadcasting listens to the broadcast channel and sends the message to all clients
func StartBroadcasting() {
	for {
		msg := <-broadcast
		mutex.Lock()
		jsonData, err := json.Marshal(msg)
		if err != nil {
			// Handle error in JSON marshaling
			mutex.Unlock()
			continue // Skip sending this message
		}

		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, jsonData)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}
