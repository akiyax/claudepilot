package handler

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/akiyax/claudepilot/daemon/internal/claude"
)

// mockTransport captures sent messages for testing.
type mockTransport struct {
	messages []claude.WSMessage
	mu       sync.Mutex
	connected bool
}

func (m *mockTransport) Start() error { return nil }
func (m *mockTransport) Send(msg claude.WSMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockTransport) OnMessage(func(msg claude.WSMessage)) {}
func (m *mockTransport) Close() error                         { return nil }
func (m *mockTransport) IsConnected() bool                    { return m.connected }

func (m *mockTransport) lastMessage() claude.WSMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return claude.WSMessage{}
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockTransport) findMessageByType(msgType string) (claude.WSMessage, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, msg := range m.messages {
		if msg.Type == msgType {
			return msg, true
		}
	}
	return claude.WSMessage{}, false
}

func newTestHandler() *Handler {
	tp := &mockTransport{connected: true}
	return NewHandler(tp, "test-daemon", "test-version")
}

func TestHandleReady(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{Type: "system.ready"})

	msg, ok := tp.findMessageByType("system.hello")
	if !ok {
		t.Fatal("expected system.hello message")
	}

	payload, ok := msg.Payload.(claude.SystemHelloPayload)
	if !ok {
		t.Fatal("payload is not SystemHelloPayload")
	}
	if payload.Version != "test-version" {
		t.Errorf("version = %q, want %q", payload.Version, "test-version")
	}
	if payload.DaemonID != "test-daemon" {
		t.Errorf("daemonId = %q", payload.DaemonID)
	}
	if payload.Mode != "lan" {
		t.Errorf("mode = %q, want %q", payload.Mode, "lan")
	}
}

func TestHandleChatMessage_NoSession(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{
		Type: "chat.message",
		Payload: map[string]any{"text": "hello"},
	})

	_, ok := tp.findMessageByType("error")
	if !ok {
		t.Fatal("expected error message when no session active")
	}
}

func TestHandleModelList(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{Type: "model.list"})

	msg, ok := tp.findMessageByType("model.list.result")
	if !ok {
		t.Fatal("expected model.list.result")
	}

	// Verify payload contains models
	data, _ := json.Marshal(msg.Payload)
	var result map[string]any
	json.Unmarshal(data, &result)

	models, ok := result["models"].([]any)
	if !ok {
		t.Fatal("expected models array")
	}
	if len(models) < 3 {
		t.Errorf("models count = %d, want >= 3", len(models))
	}
}

func TestHandleProviderList_NoManager(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{Type: "provider.list"})

	_, ok := tp.findMessageByType("error")
	if !ok {
		t.Fatal("expected error when provider manager not initialized")
	}
}

func TestHandleSessionList(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{
		Type:    "session.list",
		Payload: map[string]any{},
	})

	_, ok := tp.findMessageByType("session.list.result")
	if !ok {
		t.Fatal("expected session.list.result")
	}
}

func TestHandleAgentList(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{
		Type:    "agent.list",
		Payload: map[string]any{},
	})

	_, ok := tp.findMessageByType("agent.list.result")
	if !ok {
		t.Fatal("expected agent.list.result")
	}
}

func TestHandleUnknownMessageType(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	// Should not panic on unknown types
	h.HandleMessage(claude.WSMessage{Type: "unknown.type"})

	// No error should be sent for unknown types
	_, ok := tp.findMessageByType("error")
	if ok {
		t.Error("should not send error for unknown message types")
	}
}

func TestHandleSlashCommand_NoSession(t *testing.T) {
	h := newTestHandler()
	tp := h.transport.(*mockTransport)

	h.HandleMessage(claude.WSMessage{
		Type:    "slash.command",
		Payload: map[string]any{"command": "/help"},
	})

	_, ok := tp.findMessageByType("error")
	if !ok {
		t.Fatal("expected error when no session active for slash command")
	}
}
