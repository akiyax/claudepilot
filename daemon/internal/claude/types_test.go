package claude

import (
	"encoding/json"
	"testing"
)

func TestSystemInitParse(t *testing.T) {
	raw := `{"type":"system","subtype":"init","session_id":"sess-123","claude_code_version":"1.0.0","cwd":"/home/user/project","model":"claude-sonnet-4-6","tools":["Bash","Edit","Read"],"mcp_servers":[],"apiKeySource":"user","uuid":"uuid-1"}`

	var event SystemInit
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if event.Type != "system" {
		t.Errorf("Type = %s, want system", event.Type)
	}
	if event.SessionID != "sess-123" {
		t.Errorf("SessionID = %s, want sess-123", event.SessionID)
	}
	if event.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %s, want claude-sonnet-4-6", event.Model)
	}
	if len(event.Tools) != 3 {
		t.Errorf("Tools len = %d, want 3", len(event.Tools))
	}
}

func TestAssistantMessageParse(t *testing.T) {
	raw := `{"type":"assistant","session_id":"sess-123","uuid":"uuid-2","message":{"role":"assistant","content":[{"type":"text","text":"Hello!"},{"type":"tool_use","id":"tool-1","name":"Read","input":{"file_path":"/src/app.go"}}]}}`

	var event AssistantMessage
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if event.Type != "assistant" {
		t.Errorf("Type = %s, want assistant", event.Type)
	}
	if len(event.Message.Content) != 2 {
		t.Fatalf("Content blocks = %d, want 2", len(event.Message.Content))
	}
	if event.Message.Content[0].Text != "Hello!" {
		t.Errorf("Content[0].Text = %s, want Hello!", event.Message.Content[0].Text)
	}
	if event.Message.Content[1].Name != "Read" {
		t.Errorf("Content[1].Name = %s, want Read", event.Message.Content[1].Name)
	}
}

func TestResultMessageParse(t *testing.T) {
	raw := `{"type":"result","subtype":"success","session_id":"sess-123","uuid":"uuid-3","content":"Done!","usage":{"inputTokens":100,"outputTokens":200,"contextWindow":200000}}`

	var event ResultMessage
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if event.Subtype != "success" {
		t.Errorf("Subtype = %s, want success", event.Subtype)
	}
	if event.Usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", event.Usage.InputTokens)
	}
	if event.Usage.ContextWindow != 200000 {
		t.Errorf("ContextWindow = %d, want 200000", event.Usage.ContextWindow)
	}
}

func TestControlRequestParse(t *testing.T) {
	raw := `{"type":"control_request","request_id":"perm-1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls -la"}}}`

	var event ControlRequest
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if event.RequestID != "perm-1" {
		t.Errorf("RequestID = %s, want perm-1", event.RequestID)
	}
	if event.Request.ToolName != "Bash" {
		t.Errorf("ToolName = %s, want Bash", event.Request.ToolName)
	}
}

func TestUserInputSerialize(t *testing.T) {
	input := UserInput{
		Type: "user",
		Message: UserMessage{
			Role:    "user",
			Content: "Hello Claude",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	if parsed["type"] != "user" {
		t.Errorf("type = %v, want user", parsed["type"])
	}
	msg := parsed["message"].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("role = %v, want user", msg["role"])
	}
	if msg["content"] != "Hello Claude" {
		t.Errorf("content = %v, want Hello Claude", msg["content"])
	}
}

func TestControlResponseSerialize(t *testing.T) {
	resp := ControlResponse{
		Type: "control_response",
		Response: ControlResponseBody{
			Subtype:   "success",
			RequestID: "perm-1",
			Response: PermissionAllow{
				Behavior: "allow",
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	response := parsed["response"].(map[string]any)
	if response["subtype"] != "success" {
		t.Errorf("subtype = %v, want success", response["subtype"])
	}
}

func TestWSMessageCreation(t *testing.T) {
	msg := NewWSMessage("stream.text", StreamTextPayload{
		SessionID: "sess-1",
		Content:   "Hello",
	})
	if msg.Type != "stream.text" {
		t.Errorf("Type = %s, want stream.text", msg.Type)
	}
	if msg.Timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}
}
