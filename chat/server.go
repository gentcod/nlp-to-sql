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

// SessionState represents the explicit state machine of a WebSocket client connection
type SessionState string

const (
	StateConnected      SessionState = "CONNECTED"
	StateInitializingDB SessionState = "INITIALIZING_DB"
	StateReady          SessionState = "READY"
	StateExecutingQuery SessionState = "EXECUTING_QUERY"
	StateClosing        SessionState = "CLOSING"
	StateClosed         SessionState = "CLOSED"
)

// Message defines the format for incoming message from client.
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// Response defines the server response format (backward-compatible with optional structured Payload & State).
type Response struct {
	Type      string       `json:"type"`
	Status    string       `json:"status"`
	Message   string       `json:"message"`
	Payload   interface{}  `json:"payload,omitempty"`
	State     SessionState `json:"state,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// WebSocket server specifications.
type WebSocketServer struct {
	config    util.Config
	converter conv.Converter
	upgrader  websocket.Upgrader
	clients   map[*Client]bool
	mutex     sync.RWMutex
}

// NewWebSocketServer creates a new WebSocket server.
func NewWebSocketServer(config util.Config, converter conv.Converter) (*WebSocketServer, error) {
	return &WebSocketServer{
		config: config,
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

// HandleConnection manages a new WebSocket connection
func (srv *WebSocketServer) HandleConnection(c *gin.Context) {
	conn, err := srv.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to upgrade WebSocket connection",
			"details": err.Error(),
		})
		return
	}

	// Create a new client in the explicit CONNECTED state
	client := &Client{
		conn:      conn,
		config:    srv.config,
		converter: srv.converter,
		state:     StateConnected,
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
		c.shutdown()

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
			c.safeSend(Response{
				Type:      string(rune(msgType)),
				Status:    "error",
				Message:   fmt.Sprintf(`invalid message format: %v`, msg.Type),
				State:     c.GetState(),
				Timestamp: time.Now(),
			})
			continue
		}

		msg.Timestamp = time.Now()
		select {
		case c.receive <- msg:
		case <-c.close:
			return
		}
	}
}

// writePump handles outgoing WebSocket messages.
func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				if c.conn != nil {
					_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				}
				return
			}

			if c.conn != nil {
				if err := c.conn.WriteJSON(message); err != nil {
					log.Printf("Write error: %v", err)
					return
				}
			}

		case <-ticker.C:
			if c.conn != nil {
				if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Println("Ping error:", err)
					return
				}
			}

		case <-c.close:
			return
		}
	}
}

// processingPump handles message processing logic.
func (c *Client) processingPump() {
	defer func() {
		c.shutdown()
	}()

	for {
		select {
		case msg, ok := <-c.receive:
			if !ok {
				return
			}
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
