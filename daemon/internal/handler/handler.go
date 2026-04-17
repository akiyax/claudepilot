package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/akiyax/claudepilot/daemon/internal/claude"
	"github.com/akiyax/claudepilot/daemon/internal/transport"
)

// Handler routes WS messages to the appropriate processing logic.
// It is decoupled from the transport layer — it only depends on
// the Transport interface for sending messages back.
type Handler struct {
	transport transport.Transport
	process   *claude.ClaudeProcess
	mu        sync.Mutex
	daemonID  string
	cliVersion string
	version   string

	// Event processing
	cancel context.CancelFunc
}

// NewHandler creates a new message handler.
func NewHandler(tp transport.Transport, daemonID, version string) *Handler {
	return &Handler{
		transport: tp,
		daemonID:  daemonID,
		version:   version,
	}
}

// SetProcess sets the active Claude process and starts forwarding events.
func (h *Handler) SetProcess(p *claude.ClaudeProcess) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Cancel previous event forwarding
	if h.cancel != nil {
		h.cancel()
	}

	h.process = p

	if p != nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel = cancel
		go h.forwardEvents(ctx, p)
	}
}

// HandleMessage processes an incoming WS message from the APP.
func (h *Handler) HandleMessage(msg claude.WSMessage) {
	slog.Debug("Handling message", "type", msg.Type)

	switch msg.Type {
	case "system.ready":
		h.handleReady()

	case "chat.message":
		h.handleChatMessage(msg)

	case "chat.interrupt":
		h.handleChatInterrupt(msg)

	case "permission.respond":
		h.handlePermissionRespond(msg)

	case "question.answer":
		h.handleQuestionAnswer(msg)

	case "session.start":
		h.handleSessionStart(msg)

	case "session.list":
		// TODO: implement in Phase 2

	case "session.resume":
		// TODO: implement in Phase 2

	case "session.delete":
		// TODO: implement in Phase 2

	case "session.history":
		// TODO: implement in Phase 2

	case "agent.list":
		// TODO: implement in Phase 2

	case "agent.create":
		// TODO: implement in Phase 2

	case "model.switch":
		h.handleModelSwitch(msg)

	case "effort.switch":
		// TODO: implement

	case "mode.switch":
		// TODO: implement

	case "plan.approve":
		// TODO: implement

	case "slash.command":
		h.handleSlashCommand(msg)

	default:
		slog.Warn("Unknown message type", "type", msg.Type)
	}
}

func (h *Handler) handleReady() {
	slog.Info("Client ready")
	// Send system.hello
	h.transport.Send(claude.NewWSMessage("system.hello", claude.SystemHelloPayload{
		Version:      h.version,
		DaemonID:     h.daemonID,
		CLIVersion:   h.cliVersion,
		Mode:         "lan",
		Capabilities: []string{"chat", "permission", "question", "session", "agent", "provider", "model"},
		Commands:     []string{"/compact", "/plan", "/model", "/mode", "/memory", "/clear", "/resume", "/help"},
	}))
}

func (h *Handler) handleChatMessage(msg claude.WSMessage) {
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		h.sendError("No active session. Start a session first.")
		return
	}

	var payload claude.ChatMessagePayload
	h.unmarshalPayload(msg, &payload)

	if err := p.Send(payload.Text, nil, nil); err != nil {
		h.sendError("Failed to send message: " + err.Error())
	}
}

func (h *Handler) handleChatInterrupt(msg claude.WSMessage) {
	// TODO: implement interrupt logic (Phase 3)
	slog.Info("Chat interrupt requested", "id", msg.ID)
}

func (h *Handler) handlePermissionRespond(msg claude.WSMessage) {
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		return
	}

	var payload claude.PermissionRespondPayload
	h.unmarshalPayload(msg, &payload)

	if err := p.RespondPermission(payload.RequestID, claude.PermissionResult{
		Behavior:     payload.Behavior,
		UpdatedInput: payload.UpdatedInput,
		Message:      payload.Message,
	}); err != nil {
		slog.Error("Failed to respond to permission", "err", err)
	}

	// Send ACK
	h.transport.Send(claude.NewWSMessage("system.ack", claude.SystemAckPayload{
		RefID:    msg.ID,
		Received: true,
	}))
}

func (h *Handler) handleQuestionAnswer(msg claude.WSMessage) {
	// Question answers are also sent via RespondPermission in the stream-json protocol
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		return
	}

	// Forward as tool result
	// TODO: Implement proper AskUserQuestion response handling

	// Send ACK
	h.transport.Send(claude.NewWSMessage("system.ack", claude.SystemAckPayload{
		RefID:    msg.ID,
		Received: true,
	}))
}

func (h *Handler) handleSessionStart(msg claude.WSMessage) {
	// TODO: Full implementation in Phase 2
	// For now, just log it
	var payload claude.SessionStartPayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Session start requested", "payload", payload)
}

func (h *Handler) handleModelSwitch(msg claude.WSMessage) {
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		return
	}

	var payload claude.ModelSwitchPayload
	h.unmarshalPayload(msg, &payload)

	// Send /model command via stdin
	cmd := "/model " + payload.Model
	if err := p.Send(cmd, nil, nil); err != nil {
		h.sendError("Failed to switch model: " + err.Error())
	}
}

func (h *Handler) handleSlashCommand(msg claude.WSMessage) {
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		h.sendError("No active session")
		return
	}

	var payload claude.SlashCommandPayload
	h.unmarshalPayload(msg, &payload)

	if err := p.Send(payload.Command, nil, nil); err != nil {
		h.sendError("Failed to execute command: " + err.Error())
	}
}

// forwardEvents reads Claude process events and forwards them as WS messages.
func (h *Handler) forwardEvents(ctx context.Context, p *claude.ClaudeProcess) {
	events := p.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			h.forwardEvent(p, event)
		}
	}
}

func (h *Handler) forwardEvent(p *claude.ClaudeProcess, event claude.ClaudeEvent) {
	sessionID := p.SessionID()

	switch event.Type {
	case "system":
		var sys claude.SystemInit
		h.remarshal(event.Raw, &sys)
		if sys.SessionID != "" {
			sessionID = sys.SessionID
		}
		slog.Info("Claude system init", "sessionID", sessionID, "model", sys.Model)

	case "assistant":
		var msg claude.AssistantMessage
		h.remarshal(event.Raw, &msg)
		for _, block := range msg.Message.Content {
			switch block.Type {
			case "text":
				h.transport.Send(claude.NewWSMessage("stream.text", claude.StreamTextPayload{
					SessionID: sessionID,
					Content:   block.Text,
				}))
			case "thinking":
				h.transport.Send(claude.NewWSMessage("stream.thinking", claude.StreamThinkingPayload{
					SessionID: sessionID,
					Content:   block.Thinking,
				}))
			case "tool_use":
				h.transport.Send(claude.NewWSMessage("tool.call", claude.ToolCallPayload{
					SessionID: sessionID,
					ToolName:  block.Name,
					ToolInput: block.Input,
					ToolID:    block.ID,
				}))
			}
		}

	case "user":
		// Tool results come as user messages with tool_result content
		// Parse and forward as tool.output
		var userMsg struct {
			Message struct {
				Content []map[string]any `json:"content"`
			} `json:"message"`
		}
		h.remarshal(event.Raw, &userMsg)
		for _, block := range userMsg.Message.Content {
			if block["type"] == "tool_result" {
				toolUseID, _ := block["tool_use_id"].(string)
				content, _ := json.Marshal(block["content"])
				isError, _ := block["isError"].(bool)
				h.transport.Send(claude.NewWSMessage("tool.output", claude.ToolOutputPayload{
					SessionID: sessionID,
					ToolName:  toolUseID, // Will be resolved by APP using tool.call mapping
					Result:    string(content),
					IsError:   isError,
				}))
			}
		}

	case "result":
		var result claude.ResultMessage
		h.remarshal(event.Raw, &result)
		usage := claude.StreamUsage{}
		if result.Usage != nil {
			usage.InputTokens = result.Usage.InputTokens
			usage.OutputTokens = result.Usage.OutputTokens
			usage.TotalTokens = result.Usage.InputTokens + result.Usage.OutputTokens
			usage.ContextWindow = result.Usage.ContextWindow
			if usage.ContextWindow > 0 {
				usage.UsedPercent = usage.TotalTokens * 100 / usage.ContextWindow
			}
		}
		h.transport.Send(claude.NewWSMessage("stream.end", claude.StreamEndPayload{
			SessionID: sessionID,
			Usage:     usage,
		}))

	case "control_request":
		var req claude.ControlRequest
		h.remarshal(event.Raw, &req)
		h.transport.Send(claude.NewWSMessage("permission.request", claude.PermissionRequestPayload{
			RequestID:   req.RequestID,
			ToolName:    req.Request.ToolName,
			ToolInput:   req.Request.Input,
			ToolUseID:   req.Request.ToolUseID,
			Title:       req.Request.Title,
			DisplayText: req.Request.DisplayName,
		}))

	default:
		slog.Debug("Unhandled event type", "type", event.Type)
	}
}

// SetCLIVersion sets the detected Claude CLI version.
func (h *Handler) SetCLIVersion(v string) {
	h.cliVersion = v
}

func (h *Handler) sendError(message string) {
	h.transport.Send(claude.NewWSMessage("error", claude.ErrorPayload{
		Message: message,
	}))
}

func (h *Handler) unmarshalPayload(msg claude.WSMessage, target any) {
	if msg.Payload == nil {
		return
	}
	data, err := json.Marshal(msg.Payload)
	if err != nil {
		slog.Error("Failed to re-marshal payload", "err", err)
		return
	}
	if err := json.Unmarshal(data, target); err != nil {
		slog.Error("Failed to unmarshal payload", "type", msg.Type, "err", err)
	}
}

func (h *Handler) remarshal(raw map[string]any, target any) {
	data, err := json.Marshal(raw)
	if err != nil {
		return
	}
	json.Unmarshal(data, target)
}
