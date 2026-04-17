package transport

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// LANFileTransfer serves files over temporary HTTP for LAN transfer.
// Each file gets a unique one-time token for security.
type LANFileTransfer struct {
	port     int
	server   *http.Server
	files    map[string]*fileEntry // token -> file info
	mu       sync.RWMutex
	baseURL  string
}

type fileEntry struct {
	path      string
	createdAt time.Time
	mimeType  string
}

// NewLANFileTransfer creates a new LAN file transfer server.
func NewLANFileTransfer(port int) *LANFileTransfer {
	return &LANFileTransfer{
		port:  port,
		files: make(map[string]*fileEntry),
	}
}

// Start begins the HTTP file server.
func (t *LANFileTransfer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/file/", t.handleFile)

	t.server = &http.Server{Handler: mux}

	// Try the specified port first
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(t.port))
	if err != nil {
		// Try a random port
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return err
		}
	}

	t.port = listener.Addr().(*net.TCPAddr).Port
	t.baseURL = fmt.Sprintf("http://127.0.0.1:%d", t.port)

	go t.server.Serve(listener)

	// Cleanup old files periodically
	go t.cleanupLoop()

	return nil
}

// ServeFile registers a file and returns a one-time download URL.
func (t *LANFileTransfer) ServeFile(filePath string, mimeType string) (string, error) {
	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	token, err := generateFileToken()
	if err != nil {
		return "", err
	}

	t.mu.Lock()
	t.files[token] = &fileEntry{
		path:      filePath,
		createdAt: time.Now(),
		mimeType:  mimeType,
	}
	t.mu.Unlock()

	return fmt.Sprintf("%s/file/%s", t.baseURL, token), nil
}

// Close shuts down the file transfer server.
func (t *LANFileTransfer) Close() error {
	if t.server != nil {
		return t.server.Close()
	}
	return nil
}

// Port returns the actual port the server is listening on.
func (t *LANFileTransfer) Port() int {
	return t.port
}

func (t *LANFileTransfer) handleFile(w http.ResponseWriter, r *http.Request) {
	// Extract token from path: /file/{token}
	token := r.URL.Path[len("/file/"):]
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	t.mu.Lock()
	entry, ok := t.files[token]
	if ok {
		// One-time use: remove after first access
		delete(t.files, token)
	}
	t.mu.Unlock()

	if !ok {
		http.Error(w, "Invalid or expired file token", http.StatusNotFound)
		return
	}

	// Set content type if known
	if entry.mimeType != "" {
		w.Header().Set("Content-Type", entry.mimeType)
	}

	http.ServeFile(w, r, entry.path)
}

func (t *LANFileTransfer) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		t.mu.Lock()
		for token, entry := range t.files {
			if time.Since(entry.createdAt) > 30*time.Minute {
				delete(t.files, token)
			}
		}
		t.mu.Unlock()
	}
}

func generateFileToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
