package transport

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/akiyax/claudepilot/daemon/internal/claude"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for LAN connections
	},
}

// WSServerTransport implements Transport for V1 LAN direct connections.
type WSServerTransport struct {
	addr     string
	port     int
	server   *http.Server
	conn     *websocket.Conn
	connMu   sync.Mutex
	auth     AuthProvider
	handler  func(msg claude.WSMessage)
	daemonID string
	closed   chan struct{}
}

// NewWSServerTransport creates a new WebSocket server transport.
func NewWSServerTransport(port int, auth AuthProvider, daemonID string) *WSServerTransport {
	return &WSServerTransport{
		port:     port,
		auth:     auth,
		daemonID: daemonID,
		closed:   make(chan struct{}),
	}
}

// Port returns the actual port the server is listening on.
func (t *WSServerTransport) Port() int {
	return t.port
}

// Addr returns the address the server is listening on.
func (t *WSServerTransport) Addr() string {
	return t.addr
}

// Start begins listening for WebSocket connections.
func (t *WSServerTransport) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", t.handleWS)
	mux.HandleFunc("/pair", t.handlePair)

	t.server = &http.Server{
		Handler: mux,
	}

	// Try ports starting from the specified port
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(t.port))
	if err != nil {
		// Try auto-port
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return err
		}
	}

	t.port = listener.Addr().(*net.TCPAddr).Port
	t.addr = listener.Addr().String()

	slog.Info("WS server listening", "addr", t.addr, "port", t.port)

	go func() {
		if err := t.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("WS server error", "err", err)
		}
	}()

	return nil
}

// Send sends a message to the connected client.
func (t *WSServerTransport) Send(msg claude.WSMessage) error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.conn == nil {
		return nil // No client connected, drop the message
	}

	msg.DaemonID = t.daemonID
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}

	return t.conn.WriteJSON(msg)
}

// OnMessage registers a handler for incoming messages.
func (t *WSServerTransport) OnMessage(handler func(msg claude.WSMessage)) {
	t.handler = handler
}

// Close shuts down the transport.
func (t *WSServerTransport) Close() error {
	close(t.closed)
	if t.conn != nil {
		t.conn.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return t.server.Shutdown(ctx)
}

// IsConnected returns true if a client is connected.
func (t *WSServerTransport) IsConnected() bool {
	t.connMu.Lock()
	defer t.connMu.Unlock()
	return t.conn != nil
}

func (t *WSServerTransport) handleWS(w http.ResponseWriter, r *http.Request) {
	// Validate token
	token := r.URL.Query().Get("token")
	if t.auth != nil && !t.auth.ValidateToken(token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		slog.Warn("WS connection rejected: invalid token", "remote", r.RemoteAddr)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WS upgrade failed", "err", err)
		return
	}

	// Kick existing connection
	t.connMu.Lock()
	if t.conn != nil {
		oldConn := t.conn
		t.conn = nil
		go oldConn.Close()
		slog.Info("Kicked existing WS connection")
	}
	t.conn = conn
	t.connMu.Unlock()

	slog.Info("WS client connected", "remote", conn.RemoteAddr())

	// Read loop
	go t.readLoop(conn)
}

func (t *WSServerTransport) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	if t.auth == nil {
		http.Error(w, "Auth not configured", http.StatusInternalServerError)
		return
	}

	token, ok := t.auth.ValidatePairingCode(code)
	if !ok {
		http.Error(w, "Invalid or expired pairing code", http.StatusUnauthorized)
		return
	}

	// Return the token as JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"token":"` + token + `"}`))
	slog.Info("Pairing code validated, token issued")
}

func (t *WSServerTransport) readLoop(conn *websocket.Conn) {
	defer func() {
		t.connMu.Lock()
		if t.conn == conn {
			t.conn = nil
		}
		t.connMu.Unlock()
		conn.Close()
		slog.Info("WS client disconnected")
	}()

	for {
		var msg claude.WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("WS read error", "err", err)
			}
			return
		}

		if t.handler != nil {
			t.handler(msg)
		}
	}
}
