package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	conv "github.com/gentcod/nlp-to-sql/converter"
	mp "github.com/gentcod/nlp-to-sql/mapper"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	conn      *websocket.Conn
	converter conv.Converter
	dbConn    *sql.DB
	dbType    string
	dbName    string
	dbSchema  map[string]map[string]string
	send      chan Response
	receive   chan Message
	close     chan struct{}
	isAlive   bool
}

func (c *Client) handleDBConn(msg Message) {
	var dbData struct {
		DbType string `json:"db_type"`
		DbName string `json:"db_name"`
		DbUrl  string `json:"db_url"`
	}

	if len(msg.Payload) == 0 {
		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   `{"message": "xinvalid payload length"}`,
			Timestamp: time.Now(),
		}
		return
	}

	if err := json.Unmarshal(msg.Payload, &dbData); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)

		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   (fmt.Sprintf(`possible required missing fields, ensure the correct payload is sent containing: db_type, db_name and db_url, %v`, err)),
			Timestamp: time.Now(),
		}
		return
	}

	if dbData.DbType == "" || dbData.DbName == "" || dbData.DbUrl == "" {
		err := errors.New("database connection field(s) cannot be empty")
		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`database connection error. %v`, err),
			Timestamp: time.Now(),
		}
		return
	}

	conn, err := sql.Open(dbData.DbType, dbData.DbUrl)
	if err != nil {
		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`fialed to establish database connection. %v`, err),
			Timestamp: time.Now(),
		}
		return
	}

	mapper := mp.InitMapper(dbData.DbType)
	c.dbSchema, err = mapper.MapSchema(conn, dbData.DbName)
	if err != nil {
		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`fialed to get database context. %v`, err),
			Timestamp: time.Now(),
		}
		return
	}

	c.dbConn = conn
	c.dbType = dbData.DbType
	c.dbName = dbData.DbName

	response := Response{
		Type:      "start_response",
		Status:    "success",
		Message:   fmt.Sprintf(`successfully connected to: %v`, c.dbName),
		Timestamp: time.Now(),
	}
	c.send <- response
}

func (c *Client) handleChat(msg Message) {
	if c.dbConn == nil || c.dbType == "" || c.dbName == "" {
		err := errors.New("database chat has not been initialized")
		c.send <- Response{
			Type:      "chat_response",
			Status:    "error",
			Message:   fmt.Sprintf(`database connection error. %v`, err),
			Timestamp: time.Now(),
		}
		return
	}

	var chatReq struct {
		Question string `json:"question"`
	}

	if err := json.Unmarshal(msg.Payload, &chatReq); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)

		c.send <- Response{
			Type:      "start_response",
			Status:    "error",
			Message:   (fmt.Sprintf(`possible required missing fields, ensure the correct payload is sent. %V`, err)),
			Timestamp: time.Now(),
		}
		return
	}

	resp, err := c.converter.Convert(
		c.dbConn,
		"gemini",
		chatReq.Question,
		c.dbSchema,
	)

	if err != nil {
		c.send <- Response{
			Type:      "chat_response",
			Status:    "error",
			Message:   fmt.Sprintf(`converter error: %v`, err),
			Timestamp: time.Now(),
		}
		return
	}

	response := Response{
		Type:      "chat_response",
		Status:    "success",
		Message:   resp,
		Timestamp: time.Now(),
	}
	c.send <- response
}

func (c *Client) handleUnknownMessage(msg Message) {
	response := Response{
		Type:      "unknown",
		Status:    "error",
		Message:   fmt.Sprintf(`unknown message type: %v`, msg.Type),
		Timestamp: time.Now(),
	}
	c.send <- response
}
