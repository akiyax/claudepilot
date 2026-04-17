package transport

import "github.com/akiyax/claudepilot/daemon/internal/claude"

// Transport abstracts the communication layer between daemon and APP.
// V1: WSServerTransport (LAN direct)
// V2: WSCloudTransport (cloud relay)
type Transport interface {
	// Start begins listening for connections.
	Start() error
	// Send sends a message to the connected client.
	Send(msg claude.WSMessage) error
	// OnMessage registers a handler for incoming messages.
	OnMessage(handler func(msg claude.WSMessage))
	// Close shuts down the transport.
	Close() error
	// IsConnected returns true if a client is connected.
	IsConnected() bool
}

// AuthProvider abstracts the authentication mechanism.
type AuthProvider interface {
	// GenerateToken creates a new authentication token.
	GenerateToken() (token string, err error)
	// ValidateToken checks if a token is valid.
	ValidateToken(token string) bool
	// GeneratePairingCode creates a 6-digit pairing code.
	GeneratePairingCode() (code string, err error)
	// ValidatePairingCode checks and consumes a pairing code.
	ValidatePairingCode(code string) (token string, ok bool)
}

// FileTransfer abstracts large file transfer between daemon and APP.
type FileTransfer interface {
	// ServeFile makes a file available for download and returns the URL.
	ServeFile(filePath string) (url string, err error)
	// Close shuts down the file transfer service.
	Close() error
}
