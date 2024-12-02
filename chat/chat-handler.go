package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func (c *Client) handleDBConn(msg Message) {
	var dbConn struct {
		// Username string `json:"username"`
		DbString string `json:"token"`
	}

	if err := json.Unmarshal(msg.Payload, &dbConn); err != nil {
		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Payload:   []byte(fmt.Sprintf(`{"message": "possible required missing fields, ensure the right payload is sent in the form: %v"}`, dbConn)),
			Timestamp: time.Now(),
		}
	}

	response := Response{
		Type:      "start_response",
		Status:    "success",
		Payload:   []byte(fmt.Sprintf(`{"message": "Welcome: %v"}`, msg.Payload)),
		Timestamp: time.Now(),
	}
	c.send <- response
}

func (c *Client) handleChat(msg Message) {
	log.Printf("Received data message: %s", string(msg.Payload))

	response := Response{
		Type:      "chat_response",
		Status:    "success",
		Payload:   []byte(fmt.Sprintf(`{"message": "Hello: %v"}`, msg.Payload)),
		Timestamp: time.Now(),
	}
	c.send <- response
}

func (c *Client) handleUnknownMessage(msg Message) {
	response := Response{
		Type:      "unknown",
		Status:    "error",
		Payload:   []byte(fmt.Sprintf(`{"message": "Unknown message type: %v"}`, msg.Type)),
		Timestamp: time.Now(),
	}
	c.send <- response
}
