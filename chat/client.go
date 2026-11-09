package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	conv "github.com/gentcod/nlp-to-sql/converter"
	"github.com/gentcod/nlp-to-sql/internal/rag"
	mp "github.com/gentcod/nlp-to-sql/mapper"
	"github.com/gentcod/nlp-to-sql/util"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client with an explicit SessionState machine
type Client struct {
	conn      *websocket.Conn
	config    util.Config
	converter conv.Converter
	dbConn    *sql.DB
	dbType    string
	dbName    string
	llmType   string
	dbSchema  map[string]map[string]string
	state     SessionState
	send      chan Response
	receive   chan Message
	close     chan struct{}
	closeOnce sync.Once
	mu        sync.RWMutex
	isAlive   bool
}

// GetState returns the client's current SessionState safely
func (c *Client) GetState() SessionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Client) setState(newState SessionState) {
	c.mu.Lock()
	c.state = newState
	c.mu.Unlock()
}

// safeSend sends a response to the client without risking a panic if the connection is closing
func (c *Client) safeSend(resp Response) {
	c.mu.RLock()
	if !c.isAlive || c.state == StateClosed || c.state == StateClosing {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	select {
	case c.send <- resp:
	case <-c.close:
	default:
	}
}

// shutdown coordinates idempotent teardown of the WebSocket connection and any active DB pool
func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.state = StateClosing
		c.isAlive = false
		conn := c.conn
		dbConn := c.dbConn
		c.dbConn = nil
		c.dbSchema = nil
		c.state = StateClosed
		c.mu.Unlock()

		close(c.close)
		if conn != nil {
			_ = conn.Close()
		}
		if dbConn != nil {
			_ = dbConn.Close()
		}
	})
}

func (c *Client) handleDBConn(msg Message) {
	var dbData struct {
		DbType      string `json:"db_type"`
		DbName      string `json:"db_name"`
		DbUrl       string `json:"db_url"`
		LLMType     string `json:"llm_type"`
		UseServerDB bool   `json:"use_server_db"`
	}

	if len(msg.Payload) == 0 {
		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   `{"message": "xinvalid payload length"}`,
			State:     c.GetState(),
			Timestamp: time.Now(),
		})
		return
	}

	if err := json.Unmarshal(msg.Payload, &dbData); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)

		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`possible required missing fields, ensure the correct payload is sent containing: db_type, db_name and db_url, %v`, err),
			State:     c.GetState(),
			Timestamp: time.Now(),
		})
		return
	}

	// Optional Server DB Leg: Only bind to the server's internal database when the user explicitly opts in via use_server_db: true
	if dbData.UseServerDB {
		if c.config.DBUrl == "" || c.config.DBDriver == "" {
			c.safeSend(Response{
				Type:      "start_response",
				Status:    "error",
				Message:   "server database leg is unavailable: server database configuration is missing",
				State:     c.GetState(),
				Timestamp: time.Now(),
			})
			return
		}
		dbData.DbUrl = c.config.DBUrl
		dbData.DbType = c.config.DBDriver
		if c.config.DBName != "" {
			dbData.DbName = c.config.DBName
		} else if dbData.DbName == "" {
			dbData.DbName = "server_db"
		}
	}

	dbData.DbType = strings.TrimSpace(dbData.DbType)
	dbData.DbName = strings.TrimSpace(dbData.DbName)
	dbData.DbUrl = strings.TrimSpace(dbData.DbUrl)

	if dbData.DbType == "" || dbData.DbName == "" || dbData.DbUrl == "" {
		err := errors.New("database connection field(s) cannot be empty: please provide db_type, db_name, and db_url (or set use_server_db: true)")
		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`database connection error. %v`, err),
			State:     c.GetState(),
			Timestamp: time.Now(),
		})
		return
	}

	c.setState(StateInitializingDB)

	conn, err := sql.Open(dbData.DbType, dbData.DbUrl)
	if err != nil {
		c.setState(StateConnected)
		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`fialed to establish database connection. %v`, err),
			State:     StateConnected,
			Timestamp: time.Now(),
		})
		return
	}

	mapper := mp.InitMapper(dbData.DbType)
	if mapper == nil {
		_ = conn.Close()
		c.setState(StateConnected)
		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`unsupported database type: %v`, dbData.DbType),
			State:     StateConnected,
			Timestamp: time.Now(),
		})
		return
	}

	rawSchema, err := mapper.MapSchema(conn, dbData.DbName)
	if err != nil {
		_ = conn.Close()
		c.setState(StateConnected)
		c.safeSend(Response{
			Type:      "start_response",
			Status:    "error",
			Message:   fmt.Sprintf(`fialed to get database context. %v`, err),
			State:     StateConnected,
			Timestamp: time.Now(),
		})
		return
	}

	// Layer 1 Defense: Sanitize schema so sensitive columns are never stored in session context or passed to LLMs
	schema := util.SanitizeSchema(rawSchema)

	resolvedLLM := strings.TrimSpace(dbData.LLMType)
	if resolvedLLM == "" {
		resolvedLLM = c.config.LLMType
	}
	resolvedLLM = rag.NormalizeLLMType(resolvedLLM)

	// Safely swap out any previously opened dbConn to prevent connection pool leaks on re-"start"
	c.mu.Lock()
	oldConn := c.dbConn
	c.dbConn = conn
	c.dbType = dbData.DbType
	c.dbName = dbData.DbName
	c.llmType = resolvedLLM
	c.dbSchema = schema
	c.state = StateReady
	c.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}

	modeLabel := "custom_db"
	if dbData.UseServerDB {
		modeLabel = "server_db"
	}

	c.safeSend(Response{
		Type:      "start_response",
		Status:    "success",
		Message:   fmt.Sprintf(`successfully connected to: %v (mode: %s, llm: %s)`, dbData.DbName, modeLabel, resolvedLLM),
		State:     StateReady,
		Timestamp: time.Now(),
	})
}

func (c *Client) handleChat(msg Message) {
	c.mu.RLock()
	dbConn := c.dbConn
	dbType := c.dbType
	dbName := c.dbName
	sessionLLM := c.llmType
	dbSchema := c.dbSchema
	c.mu.RUnlock()

	if dbConn == nil || dbType == "" || dbName == "" {
		err := errors.New("database chat has not been initialized")
		c.safeSend(Response{
			Type:      "chat_response",
			Status:    "error",
			Message:   fmt.Sprintf(`database connection error. %v`, err),
			State:     c.GetState(),
			Timestamp: time.Now(),
		})
		return
	}

	var chatReq struct {
		Question string `json:"question"`
		LLMType  string `json:"llm_type"`
	}

	if err := json.Unmarshal(msg.Payload, &chatReq); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)

		c.safeSend(Response{
			Type:      "chat_response",
			Status:    "error",
			Message:   fmt.Sprintf(`possible required missing fields, ensure the correct payload is sent. %v`, err),
			State:     c.GetState(),
			Timestamp: time.Now(),
		})
		return
	}

	activeLLM := strings.TrimSpace(chatReq.LLMType)
	if activeLLM == "" {
		activeLLM = sessionLLM
	}
	if activeLLM == "" {
		activeLLM = c.config.LLMType
	}
	activeLLM = rag.NormalizeLLMType(activeLLM)

	c.setState(StateExecutingQuery)
	defer c.setState(StateReady)

	// Use DetailedConverter when available so UI and Agent clients receive SQL, rows, and latency_ms
	if detailedConv, ok := c.converter.(conv.DetailedConverter); ok {
		details, err := detailedConv.ConvertDetailed(dbConn, activeLLM, chatReq.Question, dbSchema)
		if err != nil {
			c.safeSend(Response{
				Type:      "chat_response",
				Status:    "error",
				Message:   fmt.Sprintf(`converter error: %v`, err),
				Payload:   details,
				State:     StateReady,
				Timestamp: time.Now(),
			})
			return
		}

		c.safeSend(Response{
			Type:      "chat_response",
			Status:    "success",
			Message:   details.Summary,
			Payload:   details,
			State:     StateReady,
			Timestamp: time.Now(),
		})
		return
	}

	resp, err := c.converter.Convert(
		dbConn,
		activeLLM,
		chatReq.Question,
		dbSchema,
	)

	if err != nil {
		c.safeSend(Response{
			Type:      "chat_response",
			Status:    "error",
			Message:   fmt.Sprintf(`converter error: %v`, err),
			State:     StateReady,
			Timestamp: time.Now(),
		})
		return
	}

	c.safeSend(Response{
		Type:      "chat_response",
		Status:    "success",
		Message:   resp,
		State:     StateReady,
		Timestamp: time.Now(),
	})
}

func (c *Client) handleUnknownMessage(msg Message) {
	c.safeSend(Response{
		Type:      "unknown",
		Status:    "error",
		Message:   fmt.Sprintf(`unknown message type: %v`, msg.Type),
		State:     c.GetState(),
		Timestamp: time.Now(),
	})
}
