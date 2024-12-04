package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	conv "github.com/gentcod/nlp-to-sql/converter"
	"github.com/gentcod/nlp-to-sql/token"
	"github.com/gentcod/nlp-to-sql/util"

	"github.com/gorilla/websocket"
)

// Message defines a structured incoming message format
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// Response defines a structured response message format
type Response struct {
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Advanced WebSocket server
type WebSocketServer struct {
	tokenGenerator token.Generator
	converter      conv.Converter
	upgrader       websocket.Upgrader
	clients        map[*Client]bool
	mutex          sync.RWMutex
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(config util.Config, converter conv.Converter) (*WebSocketServer, error) {
	tokenGenerator, err := token.NewPasetoGenerator(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize token generator: %v", err)
	}

	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients:        make(map[*Client]bool),
		tokenGenerator: tokenGenerator,
		converter:      converter,
	}, nil
}

// HandleConnection manages a new WebSocket connection
func (srv *WebSocketServer) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := srv.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
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
	go client.readPump()
	go client.writePump()
	go client.processingPump()
}

// readPump handles incoming WebSocket messages
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		close(c.close)
		if c.dbConn != nil {
			fmt.Println("closing db conn")
			c.dbConn.Close()
		}
	}()

	// Set read deadline to detect disconnections
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			c.isAlive = false
			break
		}

		// Add timestamp to incoming message
		msg.Timestamp = time.Now()
		c.receive <- msg
	}
}

// writePump handles outgoing WebSocket messages
func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		if c.dbConn != nil {
			fmt.Println("closing db conn")
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

// processingPump handles message processing logic
func (c *Client) processingPump() {
	defer func() {
		c.conn.Close()
		if c.dbConn != nil {
			fmt.Println("closing db conn")
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

func (srv *WebSocketServer) StartChatServer(config util.Config) error {
	connFunc := http.HandlerFunc(srv.handleConnection)

	// http.Handle("/ws", authMiddleware(srv.tokenGenerator)(connFunc))
	http.Handle("/ws", connFunc)

	log.Printf("WebSocket Server starting on %v", config.WSPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", config.WSPort), nil); err != nil {
		return fmt.Errorf("server error: %v", err)
	}
	return nil
}
