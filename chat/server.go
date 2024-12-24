package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	conv "github.com/gentcod/nlp-to-sql/converter"
	"github.com/gentcod/nlp-to-sql/util"
	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

// Message defines the format for incoming message from client.
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// Response defines the server response format.
type Response struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// WebSocket server specifications.
type WebSocketServer struct {
	converter conv.Converter
	upgrader  websocket.Upgrader
	clients   map[*Client]bool
	mutex     sync.RWMutex
}

// NewWebSocketServer creates a new WebSocket server.
func NewWebSocketServer(config util.Config, converter conv.Converter) (*WebSocketServer, error) {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:   make(map[*Client]bool),
		converter: converter,
	}, nil
}

// handleConnection manages a new WebSocket connection
func (srv *WebSocketServer) HandleConnection(c *gin.Context) {
	conn, err := srv.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to upgrade WebSocket connection",
			"details": err.Error(),
		})
		return
	}

	// Create a new client
	client := &Client{
		conn:      conn,
		converter: srv.converter,
		send:      make(chan Response, 256),
		receive:   make(chan Message, 256),
		close:     make(chan struct{}),
		isAlive:   true,
	}

	// Register client
	srv.mutex.Lock()
	srv.clients[client] = true
	srv.mutex.Unlock()

	// Start client communication loops
	go client.readPump(srv)
	go client.writePump()
	go client.processingPump()
}

// readPump handles incoming WebSocket messages.
func (c *Client) readPump(srv *WebSocketServer) {
	defer func() {
		c.conn.Close()
		if c.dbConn != nil {
			c.dbConn.Close()
		}
		c.isAlive = false

		c.conn = nil
		c.dbConn = nil
		c.dbSchema = nil
		close(c.close)
		close(c.send)
		close(c.receive)

		srv.mutex.Lock()
		delete(srv.clients, c)
		srv.mutex.Unlock()
	}()

	// Set read deadline to detect disconnections
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		msgType, reader, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
			) {
				c.isAlive = false
				break
			}
			log.Printf("WebSocket read error: %v", err)
			break
		}

		msgData, err := io.ReadAll(reader)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		err = json.Unmarshal(msgData, &msg)
		if err != nil {
			errResponse := Response{
				Type:      string(rune(msgType)),
				Status:    "error",
				Message:   fmt.Sprintf(`invalid message format: %v`, msg.Type),
				Timestamp: time.Now(),
			}

			select {
			case c.send <- errResponse:
			default:
				log.Println("Failed to send error response")
			}

			continue
		}

		msg.Timestamp = time.Now()
		c.receive <- msg
	}
}

// writePump handles outgoing WebSocket messages.
func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		if c.conn != nil {
			c.conn.Close()
		}
		if c.dbConn != nil {
			c.dbConn.Close()
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(message)
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Println("Ping error:", err)
				return
			}

		case <-c.close:
			return
		}
	}
}

// processingPump handles message processing logic.
func (c *Client) processingPump() {
	defer func() {
		if c.conn != nil {
			c.conn.Close()
		}
		if c.dbConn != nil {
			c.dbConn.Close()
		}
	}()

	for {
		select {
		case msg := <-c.receive:
			switch msg.Type {
			case "start":
				c.handleDBConn(msg)
			case "chat":
				c.handleChat(msg)
			default:
				c.handleUnknownMessage(msg)
			}

		case <-c.close:
			return
		}
	}
}
