package transport

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// TokenAuth implements AuthProvider with hex tokens and 6-digit pairing codes.
type TokenAuth struct {
	mu           sync.Mutex
	tokens       map[string]time.Time // token -> expiry
	pairingCodes map[string]*pairingEntry
}

type pairingEntry struct {
	code      string
	createdAt time.Time
}

// NewTokenAuth creates a new TokenAuth instance.
func NewTokenAuth() *TokenAuth {
	return &TokenAuth{
		tokens:       make(map[string]time.Time),
		pairingCodes: make(map[string]*pairingEntry),
	}
}

// GenerateToken creates a new 32-byte hex authentication token valid for 5 minutes.
func (a *TokenAuth) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	a.mu.Lock()
	a.tokens[token] = time.Now().Add(5 * time.Minute)
	a.mu.Unlock()

	slog.Info("Generated auth token", "prefix", token[:8])
	return token, nil
}

// ValidateToken checks if a token is valid and not expired.
func (a *TokenAuth) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	expiry, ok := a.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(a.tokens, token)
		return false
	}
	return true
}

// GeneratePairingCode creates a 6-digit pairing code valid for 5 minutes.
func (a *TokenAuth) GeneratePairingCode() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Generate a 6-digit code
	num := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	code := num % 1000000

	a.mu.Lock()
	codeStr := padCode(code)
	a.pairingCodes[codeStr] = &pairingEntry{
		code:      codeStr,
		createdAt: time.Now(),
	}
	a.mu.Unlock()

	slog.Info("Generated pairing code", "code", codeStr)
	return codeStr, nil
}

// ValidatePairingCode checks and consumes a pairing code, returning a new token.
func (a *TokenAuth) ValidatePairingCode(code string) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.pairingCodes[code]
	if !ok {
		return "", false
	}
	if time.Since(entry.createdAt) > 5*time.Minute {
		delete(a.pairingCodes, code)
		return "", false
	}

	// Consume the code
	delete(a.pairingCodes, code)

	// Generate a new token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", false
	}
	token := hex.EncodeToString(bytes)
	a.tokens[token] = time.Now().Add(24 * time.Hour) // Longer expiry for paired tokens

	slog.Info("Pairing code validated", "code", code, "tokenPrefix", token[:8])
	return token, true
}

func padCode(code int) string {
	s := strconv.Itoa(code)
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}
