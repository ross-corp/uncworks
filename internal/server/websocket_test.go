package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialHub creates an httptest server serving the hub's HandleWebSocket, connects
// a gorilla/websocket client, and returns the client conn plus a cleanup func.
func dialHub(t *testing.T, hub *WebSocketHub) (*websocket.Conn, func()) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}

	return conn, func() {
		_ = conn.Close()
		srv.Close()
	}
}

// subscribe sends a subscribe message for the given agentRunID over the conn.
func subscribe(t *testing.T, conn *websocket.Conn, agentRunID string) {
	t.Helper()
	msg := WSMessage{Type: "subscribe", AgentRunID: agentRunID}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("subscribe write: %v", err)
	}
}

// unsubscribe sends an unsubscribe message for the given agentRunID.
func unsubscribe(t *testing.T, conn *websocket.Conn, agentRunID string) {
	t.Helper()
	msg := WSMessage{Type: "unsubscribe", AgentRunID: agentRunID}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("unsubscribe write: %v", err)
	}
}

// readWSMessage reads a single WSMessage from the conn with a timeout.
func readWSMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) (WSMessage, bool) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var msg WSMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return msg, false
	}
	return msg, true
}

// waitForClient polls until the hub has the expected number of registered clients.
func waitForClient(t *testing.T, hub *WebSocketHub, count int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		hub.mu.RLock()
		n := len(hub.clients)
		hub.mu.RUnlock()
		if n == count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	hub.mu.RLock()
	t.Fatalf("expected %d clients, got %d", count, len(hub.clients))
	hub.mu.RUnlock()
}

func TestWebSocketHub_NewHub(t *testing.T) {
	hub := NewWebSocketHub()
	if hub == nil {
		t.Fatal("NewWebSocketHub returned nil")
	}
	if hub.clients == nil {
		t.Fatal("clients map is nil")
	}
	if len(hub.clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(hub.clients))
	}
}

func TestWebSocketHub_ClientRegistration(t *testing.T) {
	hub := NewWebSocketHub()

	conn1, cleanup1 := dialHub(t, hub)
	defer cleanup1()
	waitForClient(t, hub, 1)

	conn2, cleanup2 := dialHub(t, hub)
	defer cleanup2()
	waitForClient(t, hub, 2)

	// Close first client; hub should remove it.
	_ = conn1.Close()
	waitForClient(t, hub, 1)

	// Close second client; hub should be empty.
	_ = conn2.Close()
	waitForClient(t, hub, 0)
}

func TestWebSocketHub_SubscribeBroadcast(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	defer cleanup()
	waitForClient(t, hub, 1)

	subscribe(t, conn, "run-123")
	// Give the server goroutine time to process the subscribe.
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast("run-123", map[string]string{"status": "running"})

	msg, ok := readWSMessage(t, conn, 2*time.Second)
	if !ok {
		t.Fatal("expected to receive broadcast message")
	}
	if msg.Type != "event" {
		t.Fatalf("expected type 'event', got %q", msg.Type)
	}
	if msg.AgentRunID != "run-123" {
		t.Fatalf("expected agentRunId 'run-123', got %q", msg.AgentRunID)
	}

	var payload map[string]string
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["status"] != "running" {
		t.Fatalf("expected status 'running', got %q", payload["status"])
	}
}

func TestWebSocketHub_UnsubscribedClientDoesNotReceive(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	defer cleanup()
	waitForClient(t, hub, 1)

	// Client is connected but not subscribed to "run-999".
	hub.Broadcast("run-999", map[string]string{"status": "done"})

	_, ok := readWSMessage(t, conn, 200*time.Millisecond)
	if ok {
		t.Fatal("unsubscribed client should not receive broadcast")
	}
}

func TestWebSocketHub_Unsubscribe(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	defer cleanup()
	waitForClient(t, hub, 1)

	subscribe(t, conn, "run-abc")
	time.Sleep(50 * time.Millisecond)

	unsubscribe(t, conn, "run-abc")
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast("run-abc", map[string]string{"status": "done"})

	_, ok := readWSMessage(t, conn, 200*time.Millisecond)
	if ok {
		t.Fatal("client should not receive broadcast after unsubscribe")
	}
}

func TestWebSocketHub_MultipleClientsOnSameRun(t *testing.T) {
	hub := NewWebSocketHub()

	conn1, cleanup1 := dialHub(t, hub)
	defer cleanup1()
	conn2, cleanup2 := dialHub(t, hub)
	defer cleanup2()
	waitForClient(t, hub, 2)

	subscribe(t, conn1, "run-shared")
	subscribe(t, conn2, "run-shared")
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast("run-shared", map[string]string{"step": "1"})

	msg1, ok1 := readWSMessage(t, conn1, 2*time.Second)
	msg2, ok2 := readWSMessage(t, conn2, 2*time.Second)
	if !ok1 || !ok2 {
		t.Fatalf("both clients should receive the broadcast: client1=%v client2=%v", ok1, ok2)
	}
	if msg1.AgentRunID != "run-shared" || msg2.AgentRunID != "run-shared" {
		t.Fatal("broadcast agentRunID mismatch")
	}
}

func TestWebSocketHub_BroadcastOnlyToSubscribedRun(t *testing.T) {
	hub := NewWebSocketHub()

	conn1, cleanup1 := dialHub(t, hub)
	defer cleanup1()
	conn2, cleanup2 := dialHub(t, hub)
	defer cleanup2()
	waitForClient(t, hub, 2)

	subscribe(t, conn1, "run-A")
	subscribe(t, conn2, "run-B")
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast("run-A", map[string]string{"for": "A"})

	msg, ok := readWSMessage(t, conn1, 2*time.Second)
	if !ok {
		t.Fatal("client1 should receive run-A broadcast")
	}
	if msg.AgentRunID != "run-A" {
		t.Fatalf("expected agentRunId 'run-A', got %q", msg.AgentRunID)
	}

	_, ok = readWSMessage(t, conn2, 200*time.Millisecond)
	if ok {
		t.Fatal("client2 should not receive run-A broadcast (subscribed to run-B)")
	}
}

func TestWebSocketHub_ClientDisconnectionCleanup(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	waitForClient(t, hub, 1)

	subscribe(t, conn, "run-cleanup")
	time.Sleep(50 * time.Millisecond)

	// Verify a subscription exists on the server side.
	hub.mu.RLock()
	found := false
	for _, subs := range hub.clients {
		if subs["run-cleanup"] {
			found = true
			break
		}
	}
	hub.mu.RUnlock()
	if !found {
		t.Fatal("subscription should be present before disconnect")
	}

	// Disconnect the client.
	cleanup()
	waitForClient(t, hub, 0)

	// Broadcast after disconnect should not panic.
	hub.Broadcast("run-cleanup", map[string]string{"status": "orphaned"})
}

func TestWebSocketHub_SequentialBroadcasts(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	defer cleanup()
	waitForClient(t, hub, 1)

	subscribe(t, conn, "run-seq")
	time.Sleep(50 * time.Millisecond)

	const n = 20
	for i := 0; i < n; i++ {
		hub.Broadcast("run-seq", map[string]int{"i": i})
	}

	received := 0
	for {
		_, ok := readWSMessage(t, conn, 500*time.Millisecond)
		if !ok {
			break
		}
		received++
	}
	if received != n {
		t.Fatalf("expected %d messages, got %d", n, received)
	}
}

func TestWebSocketHub_MultipleSubscriptions(t *testing.T) {
	hub := NewWebSocketHub()

	conn, cleanup := dialHub(t, hub)
	defer cleanup()
	waitForClient(t, hub, 1)

	subscribe(t, conn, "run-X")
	subscribe(t, conn, "run-Y")
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast("run-X", map[string]string{"run": "X"})
	hub.Broadcast("run-Y", map[string]string{"run": "Y"})

	msg1, ok1 := readWSMessage(t, conn, 2*time.Second)
	msg2, ok2 := readWSMessage(t, conn, 2*time.Second)
	if !ok1 || !ok2 {
		t.Fatal("client should receive broadcasts for both subscribed runs")
	}
	if msg1.AgentRunID != "run-X" {
		t.Fatalf("first message agentRunId: want 'run-X', got %q", msg1.AgentRunID)
	}
	if msg2.AgentRunID != "run-Y" {
		t.Fatalf("second message agentRunId: want 'run-Y', got %q", msg2.AgentRunID)
	}
}
