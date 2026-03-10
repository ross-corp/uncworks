package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev; restrict in production
	},
}

// WebSocketHub manages WebSocket connections for real-time event streaming.
type WebSocketHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]map[string]bool // conn -> set of subscribed agent run IDs
}

// NewWebSocketHub creates a new WebSocket hub.
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]map[string]bool),
	}
}

// WSMessage is the message format for WebSocket communication.
type WSMessage struct {
	Type       string          `json:"type"`
	AgentRunID string          `json:"agentRunId,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// HandleWebSocket handles WebSocket upgrade and message routing.
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	h.mu.Lock()
	h.clients[conn] = make(map[string]bool)
	h.mu.Unlock()

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "subscribe":
			h.mu.Lock()
			h.clients[conn][msg.AgentRunID] = true
			h.mu.Unlock()
		case "unsubscribe":
			h.mu.Lock()
			delete(h.clients[conn], msg.AgentRunID)
			h.mu.Unlock()
		}
	}
}

// Broadcast sends an event to all WebSocket clients subscribed to the given agent run.
func (h *WebSocketHub) Broadcast(agentRunID string, event any) {
	data, err := json.Marshal(WSMessage{
		Type:       "event",
		AgentRunID: agentRunID,
		Payload:    mustMarshal(event),
	})
	if err != nil {
		log.Printf("WebSocket marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn, subs := range h.clients {
		if subs[agentRunID] {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("WebSocket payload marshal error: %v", err)
		return json.RawMessage(`{}`)
	}
	return b
}
