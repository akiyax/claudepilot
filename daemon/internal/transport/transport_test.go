package transport

import (
	"testing"
)

func TestTokenAuth_GenerateAndValidate(t *testing.T) {
	auth := NewTokenAuth()

	token, err := auth.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("token length = %d, want 64", len(token))
	}
	if !auth.ValidateToken(token) {
		t.Error("token should be valid")
	}
	if auth.ValidateToken("invalid") {
		t.Error("invalid token should not be valid")
	}
	if auth.ValidateToken("") {
		t.Error("empty token should not be valid")
	}
}

func TestTokenAuth_PairingCode(t *testing.T) {
	auth := NewTokenAuth()

	code, err := auth.GeneratePairingCode()
	if err != nil {
		t.Fatalf("GeneratePairingCode error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code length = %d, want 6", len(code))
	}

	// Validate the code and get a token
	token, ok := auth.ValidatePairingCode(code)
	if !ok {
		t.Error("pairing code should be valid")
	}
	if token == "" {
		t.Error("should return a token")
	}

	// Code should be consumed
	_, ok = auth.ValidatePairingCode(code)
	if ok {
		t.Error("pairing code should be consumed after first use")
	}

	// The returned token should be valid
	if !auth.ValidateToken(token) {
		t.Error("returned token should be valid")
	}
}

func TestTokenAuth_InvalidPairingCode(t *testing.T) {
	auth := NewTokenAuth()

	_, ok := auth.ValidatePairingCode("000000")
	if ok {
		t.Error("non-existent code should not be valid")
	}
}

func TestWSServerTransport_NotConnected(t *testing.T) {
	tr := NewWSServerTransport(0, nil, "test-daemon")
	if tr.IsConnected() {
		t.Error("should not be connected initially")
	}
}

func TestPadCode(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "000000"},
		{1, "000001"},
		{123456, "123456"},
		{999999, "999999"},
	}
	for _, tt := range tests {
		got := padCode(tt.input)
		if got != tt.want {
			t.Errorf("padCode(%d) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
