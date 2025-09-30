package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
)

// Server represents the WebSocket server
type Server struct {
	addr      string
	upgrader  websocket.Upgrader
	sessions  map[string]*ClientSession
	mu        sync.RWMutex
	broadcast chan Message
}

// ClientSession represents a client WebSocket session
type ClientSession struct {
	ID           string
	Conn         *websocket.Conn
	ClaudeClient *claude.Client
	TmuxManager  *claude.TmuxManager
	SendChan     chan Message
	Context      context.Context
	Cancel       context.CancelFunc
	mu           sync.Mutex
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"sessionId,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// IntentRequest from client
type IntentRequest struct {
	Type      string `json:"type"`
	Intent    string `json:"intent"`
	SessionID string `json:"sessionId"`
}

// IntentResponse to client
type IntentResponse struct {
	Type         string                 `json:"type"`
	SessionID    string                 `json:"sessionId"`
	Intent       string                 `json:"intent"`
	SliceType    string                 `json:"sliceType"`
	Action       string                 `json:"action"`
	Requirements map[string]interface{} `json:"requirements"`
	RawResponse  string                 `json:"rawResponse,omitempty"`
	Status       string                 `json:"status"`
	Timestamp    int64                  `json:"timestamp"`
}

// StreamChunk for streaming responses
type StreamChunk struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Chunk     string `json:"chunk"`
	Done      bool   `json:"done"`
	Timestamp int64  `json:"timestamp"`
}

// NewServer creates a new WebSocket server
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for demo
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		sessions:  make(map[string]*ClientSession),
		broadcast: make(chan Message, 100),
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// Start broadcast handler
	go s.handleBroadcast()

	// Setup HTTP routes
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/", s.serveHome)

	log.Printf("🚀 WebSocket server starting on %s", s.addr)
	return http.ListenAndServe(s.addr, nil)
}

// HandleWebSocket handles WebSocket connections (exported for testing)
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.handleWebSocket(w, r)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create new session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called on all paths

	// Create Claude client for this session
	claudeConfig := &claude.ClientConfig{
		SessionName: fmt.Sprintf("ws-%s", sessionID[:8]),
		Timeout:     30 * time.Second,
		UseFallback: false,
	}

	claudeClient, err := claude.NewClient(ctx, claudeConfig)
	if err != nil {
		log.Printf("Failed to create Claude client: %v", err)
		conn.Close()
		return
	}

	session := &ClientSession{
		ID:           sessionID,
		Conn:         conn,
		ClaudeClient: claudeClient,
		SendChan:     make(chan Message, 100),
		Context:      ctx,
		Cancel:       cancel,
	}

	// Register session
	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// Send welcome message
	welcome := Message{
		Type:      "connected",
		SessionID: sessionID,
		Message:   "Connected to O-RAN Network Slicing Claude Service",
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}
	session.SendMessage(welcome)

	// Start handlers
	go session.handleWrite()
	go session.handleRead(s)

	log.Printf("✅ New WebSocket client connected: %s", sessionID)
}

// handleRead handles reading from WebSocket
func (c *ClientSession) handleRead(server *Server) {
	defer func() {
		c.Cancel()
		c.Conn.Close()
		if c.ClaudeClient != nil {
			c.ClaudeClient.Cleanup(c.Context)
		}
		server.removeSession(c.ID)
		log.Printf("❌ Client disconnected: %s", c.ID)
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var req IntentRequest
		err := c.Conn.ReadJSON(&req)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process intent
		go c.processIntent(req)
	}
}

// handleWrite handles writing to WebSocket
func (c *ClientSession) handleWrite() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendChan:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.Context.Done():
			return
		}
	}
}

// processIntent processes an intent request
func (c *ClientSession) processIntent(req IntentRequest) {
	log.Printf("📝 Processing intent for session %s: %s", c.ID, req.Intent)

	// Send processing status
	c.SendMessage(Message{
		Type:      "processing",
		SessionID: c.ID,
		Message:   "Processing your intent...",
		Status:    "processing",
		Timestamp: time.Now().Unix(),
	})

	// Process with Claude
	intentReq := &claude.IntentRequest{
		Text: req.Intent,
	}

	response, err := c.ClaudeClient.ProcessIntent(c.Context, intentReq)
	if err != nil {
		c.SendMessage(Message{
			Type:      "error",
			SessionID: c.ID,
			Message:   fmt.Sprintf("Error processing intent: %v", err),
			Status:    "error",
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// Build response
	intentResp := IntentResponse{
		Type:      "intent_response",
		SessionID: c.ID,
		Intent:    req.Intent,
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}

	if response.ParsedIntent != nil {
		intentResp.SliceType = response.ParsedIntent.SliceType
		intentResp.Action = response.ActionType
		intentResp.Requirements = map[string]interface{}{
			"throughput":  response.ParsedIntent.Requirements.Throughput,
			"latency":     response.ParsedIntent.Requirements.Latency,
			"reliability": response.ParsedIntent.Requirements.Reliability,
		}
	}

	if response.Response != "" {
		intentResp.RawResponse = response.Response
	}

	// Send response
	if data, err := json.Marshal(intentResp); err == nil {
		var msg Message
		json.Unmarshal(data, &msg)
		c.SendMessage(msg)
	}

	log.Printf("✅ Intent processed for session %s: SliceType=%s, Action=%s",
		c.ID, intentResp.SliceType, intentResp.Action)
}

// SendMessage sends a message to the client
func (c *ClientSession) SendMessage(msg Message) {
	select {
	case c.SendChan <- msg:
	case <-time.After(time.Second):
		log.Printf("Send timeout for session %s", c.ID)
	}
}

// handleBroadcast handles broadcasting messages
func (s *Server) handleBroadcast() {
	for {
		msg := <-s.broadcast
		s.mu.RLock()
		for _, session := range s.sessions {
			select {
			case session.SendChan <- msg:
			default:
				close(session.SendChan)
				delete(s.sessions, session.ID)
			}
		}
		s.mu.RUnlock()
	}
}

// removeSession removes a session
func (s *Server) removeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// HandleHealth handles health check (exported for testing)
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	s.handleHealth(w, r)
}

// handleHealth handles health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	activeSessions := len(s.sessions)
	s.mu.RUnlock()

	health := map[string]interface{}{
		"status":         "healthy",
		"activeSessions": activeSessions,
		"timestamp":      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// ServeHome serves the HTML frontend (exported for testing)
func (s *Server) ServeHome(w http.ResponseWriter, r *http.Request) {
	s.serveHome(w, r)
}

// serveHome serves the HTML frontend
func (s *Server) serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

// BroadcastMessage broadcasts a message to all clients
func (s *Server) BroadcastMessage(msg Message) {
	s.broadcast <- msg
}

// GetActiveSessions returns the number of active sessions
func (s *Server) GetActiveSessions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}