package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSHub manages websocket connections
type WSHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
	upgrader websocket.Upgrader
}

// NewHub creates a new websocket hub
func NewHub() *WSHub {
	// Get allowed origins from environment
	allowedOrigins := getAllowedOrigins()

	return &WSHub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// For MVP, allow all local development origins
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // Allow same-origin requests
				}
				// Allow localhost and 127.0.0.1 on any port
				if strings.HasPrefix(origin, "http://localhost") ||
				   strings.HasPrefix(origin, "http://127.0.0.1") ||
				   strings.HasPrefix(origin, "http://0.0.0.0") ||
				   strings.HasPrefix(origin, "http://[::1]") {
					return true
				}
				// Check against explicit allowed origins
				return validateOrigin(r, allowedOrigins)
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// getAllowedOrigins returns allowed origins from environment
func getAllowedOrigins() []string {
	originsStr := env("BG_WS_ALLOWED_ORIGINS", "")
	if originsStr == "" {
		// Default: allow localhost and common local network addresses
		return []string{
			"http://localhost",
			"http://localhost:3000",
			"http://127.0.0.1",
			"http://127.0.0.1:3000",
			"http://0.0.0.0",
			"http://[::1]",
		}
	}

	// Parse comma-separated origins
	return strings.Split(originsStr, ",")
}

// validateOrigin checks if the request origin is allowed
func validateOrigin(r *http.Request, allowedOrigins []string) bool {
	// Get the origin header
	origin := r.Header.Get("Origin")
	if origin == "" {
		// For same-origin requests, Origin header may be empty
		// In this case, check the Host header
		host := r.Host
		for _, allowed := range allowedOrigins {
			if strings.Contains(allowed, host) || strings.Contains(host, strings.TrimPrefix(allowed, "http://")) {
				return true
			}
		}
		return false
	}

	// Check against allowed origins
	for _, allowed := range allowedOrigins {
		if origin == allowed || strings.HasPrefix(origin, allowed) {
			return true
		}
	}

	return false
}

// HandleWS handles websocket connection upgrades
func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Log the incoming request for debugging
	log.Printf("WebSocket connection attempt from %s, Origin: %s", r.RemoteAddr, r.Header.Get("Origin"))

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	log.Printf("WebSocket connection established from %s", r.RemoteAddr)

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Send welcome message
	_ = h.SendTo(conn, WSMessage{
		Type: "connected",
		TS:   time.Now().UTC().Format(time.RFC3339),
	})

	// Keep connection alive and handle incoming messages
	go h.readLoop(conn)

	// Set up connection ping/pong
	go h.pingLoop(conn)
}

// readLoop reads messages from the websocket connection
func (h *WSHub) readLoop(conn *websocket.Conn) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		_ = conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// pingLoop sends periodic ping messages
func (h *WSHub) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		_, exists := h.clients[conn]
		h.mu.RUnlock()

		if !exists {
			return
		}

		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			return
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WSHub) Broadcast(msgType string, data map[string]interface{}) {
	msg := WSMessage{
		Type: msgType,
		Data: data,
		TS:   time.Now().UTC().Format(time.RFC3339),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		_ = conn.WriteMessage(websocket.TextMessage, b)
	}
}

// SendTo sends a message to a specific client
func (h *WSHub) SendTo(conn *websocket.Conn, msg WSMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, b)
}

// GetClientCount returns the number of connected clients
func (h *WSHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
