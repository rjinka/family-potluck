package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("Hub clients map is nil")
	}
	if hub.broadcast == nil {
		t.Error("Hub broadcast channel is nil")
	}
	if hub.register == nil {
		t.Error("Hub register channel is nil")
	}
	if hub.unregister == nil {
		t.Error("Hub unregister channel is nil")
	}
}

func TestHubIntegration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a test server that uses the hub
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r)
	}))
	defer s.Close()

	// Convert http URL to ws URL
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("Failed to connect to websocket: %v", err)
	}
	defer ws.Close()

	// Give some time for registration
	time.Sleep(50 * time.Millisecond)

	// Check if client is registered
	// We can't access hub.clients safely without a mutex or exposing a method,
	// but we can verify functionality by broadcasting.

	message := []byte("hello world")
	hub.Broadcast(message)

	// Read message from websocket
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(p) != string(message) {
		t.Errorf("Expected message %s, got %s", message, p)
	}
}
