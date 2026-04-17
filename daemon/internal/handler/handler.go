package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"

	"github.com/akiyax/claudepilot/daemon/internal/agent"
	"github.com/akiyax/claudepilot/daemon/internal/claude"
	"github.com/akiyax/claudepilot/daemon/internal/provider"
	"github.com/akiyax/claudepilot/daemon/internal/session"
	"github.com/akiyax/claudepilot/daemon/internal/transport"
)

// Handler routes WS messages to the appropriate processing logic.
type Handler struct {
	transport transport.Transport
	process   *claude.ClaudeProcess
	mu        sync.Mutex
	daemonID  string
	cliVersion string
	version   string

	// Cancel event forwarding
	cancel context.CancelFunc

	// Managers
	agentMgr    *agent.Manager
	sessionMgr  *session.Reader
	providerMgr *provider.Manager
}

// NewHandler creates a new message handler.
func NewHandler(tp transport.Transport, daemonID, version string) *Handler {
	return &Handler{
		transport:  tp,
		daemonID:   daemonID,
		version:    version,
		agentMgr:   agent.NewManager(),
		sessionMgr: session.NewReader(),
	}
}

// InitProviderManager initializes the provider manager (may fail on bad config).
func (h *Handler) InitProviderManager() error {
	mgr, err := provider.NewManager()
	if err != nil {
		return err
	}
	h.providerMgr = mgr
	return nil
}

// SetProcess sets the active Claude process and starts forwarding events.
func (h *Handler) SetProcess(p *claude.ClaudeProcess) {
	h.mu.Lock()
	defer h.mu.Unlock()

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

	// Chat
	case "chat.message":
		h.handleChatMessage(msg)
	case "chat.interrupt":
		h.handleChatInterrupt(msg)

	// Permissions & Questions
	case "permission.respond":
		h.handlePermissionRespond(msg)
	case "question.answer":
		h.handleQuestionAnswer(msg)

	// Session management
	case "session.start":
		h.handleSessionStart(msg)
	case "session.list":
		h.handleSessionList(msg)
	case "session.resume", "session.switch":
		h.handleSessionResume(msg)
	case "session.delete":
		h.handleSessionDelete(msg)
	case "session.history":
		h.handleSessionHistory(msg)

	// Agent management
	case "agent.list":
		h.handleAgentList(msg)
	case "agent.get":
		h.handleAgentGet(msg)
	case "agent.create":
		h.handleAgentCreate(msg)
	case "agent.update":
		h.handleAgentUpdate(msg)
	case "agent.delete":
		h.handleAgentDelete(msg)

	// Provider management
	case "provider.list":
		h.handleProviderList(msg)
	case "provider.add":
		h.handleProviderAdd(msg)
	case "provider.remove":
		h.handleProviderRemove(msg)
	case "provider.switch":
		h.handleProviderSwitch(msg)

	// Model & Effort & Mode
	case "model.switch":
		h.handleModelSwitch(msg)
	case "model.list":
		h.handleModelList(msg)
	case "effort.switch":
		h.handleEffortSwitch(msg)
	case "mode.switch":
		h.handleModeSwitch(msg)

	// Plan
	case "plan.approve":
		h.handlePlanApprove(msg)

	// Slash command
	case "slash.command":
		h.handleSlashCommand(msg)

	default:
		slog.Warn("Unknown message type", "type", msg.Type)
	}
}

// ─── System ────────────────────────────────────────────

func (h *Handler) handleReady() {
	slog.Info("Client ready")
	h.transport.Send(claude.NewWSMessage("system.hello", claude.SystemHelloPayload{
		Version:      h.version,
		DaemonID:     h.daemonID,
		CLIVersion:   h.cliVersion,
		Mode:         "lan",
		Capabilities: []string{"chat", "permission", "question", "session", "agent", "provider", "model"},
		Commands:     []string{"/compact", "/plan", "/model", "/mode", "/memory", "/clear", "/resume", "/help"},
	}))
}

// ─── Chat ──────────────────────────────────────────────

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
	slog.Info("Chat interrupt requested", "id", msg.ID)
	// TODO: Phase 3 — implement interrupt logic
}

// ─── Permissions & Questions ───────────────────────────

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
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		return
	}

	// Send ACK
	h.transport.Send(claude.NewWSMessage("system.ack", claude.SystemAckPayload{
		RefID:    msg.ID,
		Received: true,
	}))
}

// ─── Session management ───────────────────────────────

func (h *Handler) handleSessionStart(msg claude.WSMessage) {
	var payload claude.SessionStartPayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Session start requested", "projectDir", payload.ProjectDir, "agent", payload.AgentName, "model", payload.Model)

	// Build process options
	opts := claude.ProcessOptions{
		WorkDir:  payload.ProjectDir,
		Model:    payload.Model,
		AgentName: payload.AgentName,
		Effort:   payload.Effort,
		PermissionMode: payload.PermissionMode,
	}

	// Default workDir to home directory
	if opts.WorkDir == "" {
		home, _ := os.UserHomeDir()
		opts.WorkDir = home
	}

	// Provider env vars
	if h.providerMgr != nil && payload.Provider != "" && payload.Provider != "default" {
		if err := h.providerMgr.Switch(payload.Provider); err != nil {
			slog.Warn("Failed to switch provider", "provider", payload.Provider, "err", err)
		}
	}
	if h.providerMgr != nil {
		opts.Env = h.providerMgr.EnvVars()
	}

	// Stop existing process if any
	h.mu.Lock()
	if h.process != nil && h.process.Alive() {
		slog.Info("Stopping existing session before starting new one")
		old := h.process
		h.mu.Unlock()
		old.Close()
		h.mu.Lock()
	}
	h.mu.Unlock()

	// Start new process
	proc, err := claude.NewProcess(context.Background(), opts)
	if err != nil {
		h.sendError("Failed to start Claude Code: " + err.Error())
		return
	}

	h.SetProcess(proc)
	slog.Info("Claude Code process started", "pid", proc.PID())
}

func (h *Handler) handleSessionList(msg claude.WSMessage) {
	var payload claude.SessionListPayload
	h.unmarshalPayload(msg, &payload)

	sessions, err := h.sessionMgr.ListSessions(payload.ProjectDir)
	if err != nil {
		h.sendError("Failed to list sessions: " + err.Error())
		return
	}

	items := make([]claude.SessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = claude.SessionItem{
			ID:           s.ID,
			ProjectDir:   s.ProjectDir,
			Summary:      s.Summary,
			MessageCount: s.MessageCount,
			ModifiedAt:   s.ModifiedAt.Unix(),
		}
	}

	h.transport.Send(claude.NewWSMessage("session.list.result", claude.SessionListResultPayload{
		Sessions: items,
	}))
}

func (h *Handler) handleSessionResume(msg claude.WSMessage) {
	var payload claude.SessionResumePayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Session resume requested", "sessionId", payload.SessionID)

	// Stop existing process
	h.mu.Lock()
	if h.process != nil && h.process.Alive() {
		old := h.process
		h.mu.Unlock()
		old.Close()
		h.mu.Lock()
	}
	h.mu.Unlock()

	// Resume session
	home, _ := os.UserHomeDir()
	opts := claude.ProcessOptions{
		WorkDir:    home,
		SessionID:  payload.SessionID,
	}

	if h.providerMgr != nil {
		opts.Env = h.providerMgr.EnvVars()
	}

	proc, err := claude.NewProcess(context.Background(), opts)
	if err != nil {
		h.sendError("Failed to resume session: " + err.Error())
		return
	}

	h.SetProcess(proc)
	slog.Info("Session resumed", "sessionId", payload.SessionID)
}

func (h *Handler) handleSessionDelete(msg claude.WSMessage) {
	var payload claude.SessionDeletePayload
	h.unmarshalPayload(msg, &payload)

	if err := h.sessionMgr.DeleteSession(payload.SessionID, ""); err != nil {
		h.sendError("Failed to delete session: " + err.Error())
		return
	}

	h.transport.Send(claude.NewWSMessage("session.updated", nil))
}

func (h *Handler) handleSessionHistory(msg claude.WSMessage) {
	var payload claude.SessionHistoryPayload
	h.unmarshalPayload(msg, &payload)

	limit := payload.Limit
	if limit <= 0 {
		limit = 50
	}

	history, err := h.sessionMgr.GetSessionHistory(payload.SessionID, "", limit)
	if err != nil {
		h.sendError("Failed to get session history: " + err.Error())
		return
	}

	messages := make([]claude.HistoryMessage, len(history))
	for i, entry := range history {
		messages[i] = claude.HistoryMessage{
			Type:      entry.Type,
			Content:   entry.Content,
			Timestamp: entry.Timestamp.Unix(),
		}
	}

	h.transport.Send(claude.NewWSMessage("session.history.result", claude.SessionHistoryResultPayload{
		Messages: messages,
	}))
}

// ─── Agent management ──────────────────────────────────

func (h *Handler) handleAgentList(msg claude.WSMessage) {
	var payload claude.AgentListPayload
	h.unmarshalPayload(msg, &payload)

	agents, err := h.agentMgr.ListAgents(payload.ProjectDir)
	if err != nil {
		h.sendError("Failed to list agents: " + err.Error())
		return
	}

	items := make([]claude.AgentItem, len(agents))
	for i, a := range agents {
		items[i] = claude.AgentItem{
			Name:        a.Name,
			Description: a.Description,
			Model:       a.Model,
			Color:       a.Color,
			Source:       string(a.Source),
		}
	}

	h.transport.Send(claude.NewWSMessage("agent.list.result", claude.AgentListResultPayload{
		Agents: items,
	}))
}

func (h *Handler) handleAgentGet(msg claude.WSMessage) {
	var payload claude.AgentGetPayload
	h.unmarshalPayload(msg, &payload)

	detail, err := h.agentMgr.GetAgent(payload.Name, payload.ProjectDir)
	if err != nil {
		h.sendError("Failed to get agent: " + err.Error())
		return
	}

	h.transport.Send(claude.NewWSMessage("agent.get.result", claude.AgentGetResultPayload{
		Name:            detail.Name,
		Description:     detail.Description,
		Model:           detail.Model,
		Color:           detail.Color,
		Source:           string(detail.Source),
		Tools:           detail.Tools,
		DisallowedTools: detail.DisallowedTools,
		PermissionMode:  detail.PermissionMode,
		Effort:          detail.Effort,
		MaxTurns:        detail.MaxTurns,
		Memory:          detail.Memory,
		InitialPrompt:   detail.InitialPrompt,
		Isolation:       detail.Isolation,
		Background:      detail.Background,
		Prompt:          detail.Prompt,
	}))
}

func (h *Handler) handleAgentCreate(msg claude.WSMessage) {
	var payload claude.AgentCreatePayload
	h.unmarshalPayload(msg, &payload)

	err := h.agentMgr.CreateAgent(agent.AgentDetail{
		AgentMeta: agent.AgentMeta{
			Name:        payload.Name,
			Description: payload.Description,
			Model:       payload.Model,
			Color:       payload.Color,
		},
		Tools:           payload.Tools,
		DisallowedTools: payload.DisallowedTools,
		PermissionMode:  payload.PermissionMode,
		Effort:          payload.Effort,
		MaxTurns:        payload.MaxTurns,
		Memory:          payload.Memory,
		InitialPrompt:   payload.InitialPrompt,
		Isolation:       payload.Isolation,
		Background:      payload.Background,
		Prompt:          payload.Prompt,
	}, payload.ProjectDir)

	if err != nil {
		h.sendError("Failed to create agent: " + err.Error())
		return
	}

	h.transport.Send(claude.NewWSMessage("session.updated", nil))
	slog.Info("Agent created", "name", payload.Name)
}

func (h *Handler) handleAgentUpdate(msg claude.WSMessage) {
	var payload claude.AgentUpdatePayload
	h.unmarshalPayload(msg, &payload)

	update := agent.AgentDetail{
		AgentMeta: agent.AgentMeta{
			Name:        payload.Name,
			Description: payload.Description,
			Model:       payload.Model,
			Color:       payload.Color,
		},
		Tools:           payload.Tools,
		DisallowedTools: payload.DisallowedTools,
		PermissionMode:  payload.PermissionMode,
		Effort:          payload.Effort,
		MaxTurns:        payload.MaxTurns,
		Memory:          payload.Memory,
		InitialPrompt:   payload.InitialPrompt,
		Isolation:       payload.Isolation,
		Prompt:          payload.Prompt,
	}
	if payload.Background != nil {
		update.Background = *payload.Background
	}

	if err := h.agentMgr.UpdateAgent(payload.Name, payload.ProjectDir, update); err != nil {
		h.sendError("Failed to update agent: " + err.Error())
		return
	}

	slog.Info("Agent updated", "name", payload.Name)
}

func (h *Handler) handleAgentDelete(msg claude.WSMessage) {
	var payload claude.AgentDeletePayload
	h.unmarshalPayload(msg, &payload)

	if err := h.agentMgr.DeleteAgent(payload.Name, payload.ProjectDir); err != nil {
		h.sendError("Failed to delete agent: " + err.Error())
		return
	}

	h.transport.Send(claude.NewWSMessage("session.updated", nil))
	slog.Info("Agent deleted", "name", payload.Name)
}

// ─── Provider management ───────────────────────────────

func (h *Handler) handleProviderList(msg claude.WSMessage) {
	if h.providerMgr == nil {
		h.sendError("Provider manager not initialized")
		return
	}

	providers := h.providerMgr.List()
	active := h.providerMgr.GetActive()

	items := make([]claude.ProviderItem, len(providers))
	for i, p := range providers {
		items[i] = claude.ProviderItem{
			Name:      p.Name,
			IsDefault: p.IsDefault,
			BaseURL:   p.BaseURL,
			Model:     p.Model,
		}
	}

	h.transport.Send(claude.NewWSMessage("provider.list.result", claude.ProviderListResultPayload{
		Providers: items,
		Active:    active.Name,
	}))
}

func (h *Handler) handleProviderAdd(msg claude.WSMessage) {
	if h.providerMgr == nil {
		h.sendError("Provider manager not initialized")
		return
	}

	var payload claude.ProviderAddPayload
	h.unmarshalPayload(msg, &payload)

	if err := h.providerMgr.Add(provider.ProviderConfig{
		Name:    payload.Name,
		APIKey:  payload.APIKey,
		BaseURL: payload.BaseURL,
		Model:   payload.Model,
	}); err != nil {
		h.sendError("Failed to add provider: " + err.Error())
		return
	}

	slog.Info("Provider added", "name", payload.Name)
}

func (h *Handler) handleProviderRemove(msg claude.WSMessage) {
	if h.providerMgr == nil {
		h.sendError("Provider manager not initialized")
		return
	}

	var payload claude.ProviderRemovePayload
	h.unmarshalPayload(msg, &payload)

	if err := h.providerMgr.Remove(payload.Name); err != nil {
		h.sendError("Failed to remove provider: " + err.Error())
		return
	}

	slog.Info("Provider removed", "name", payload.Name)
}

func (h *Handler) handleProviderSwitch(msg claude.WSMessage) {
	if h.providerMgr == nil {
		h.sendError("Provider manager not initialized")
		return
	}

	var payload claude.ProviderSwitchPayload
	h.unmarshalPayload(msg, &payload)

	if err := h.providerMgr.Switch(payload.Name); err != nil {
		h.sendError("Failed to switch provider: " + err.Error())
		return
	}

	slog.Info("Provider switched", "name", payload.Name)
}

// ─── Model, Effort, Mode ──────────────────────────────

func (h *Handler) handleModelSwitch(msg claude.WSMessage) {
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p == nil || !p.Alive() {
		return
	}

	var payload claude.ModelSwitchPayload
	h.unmarshalPayload(msg, &payload)

	cmd := "/model " + payload.Model
	if err := p.Send(cmd, nil, nil); err != nil {
		h.sendError("Failed to switch model: " + err.Error())
	}
}

func (h *Handler) handleModelList(msg claude.WSMessage) {
	// Return common models
	models := []map[string]string{
		{"id": "opus", "name": "Claude Opus"},
		{"id": "sonnet", "name": "Claude Sonnet"},
		{"id": "haiku", "name": "Claude Haiku"},
	}
	h.transport.Send(claude.NewWSMessage("model.list.result", map[string]any{
		"models": models,
	}))
}

func (h *Handler) handleEffortSwitch(msg claude.WSMessage) {
	var payload claude.EffortSwitchPayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Effort switch requested", "effort", payload.Effort)

	// Send as slash command to stdin
	h.mu.Lock()
	p := h.process
	h.mu.Unlock()

	if p != nil && p.Alive() {
		cmd := "/config effort=" + payload.Effort
		p.Send(cmd, nil, nil)
	}
}

func (h *Handler) handleModeSwitch(msg claude.WSMessage) {
	var payload claude.ModeSwitchPayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Mode switch requested", "mode", payload.Mode)
	// Mode switching is handled in-memory by the daemon's permission handler
	// Future implementation: store mode and use it for permission decisions
}

func (h *Handler) handlePlanApprove(msg claude.WSMessage) {
	var payload claude.PlanApprovePayload
	h.unmarshalPayload(msg, &payload)
	slog.Info("Plan approval", "approved", payload.Approved, "planId", payload.PlanID)

	h.transport.Send(claude.NewWSMessage("system.ack", claude.SystemAckPayload{
		RefID:    msg.ID,
		Received: true,
	}))
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

// ─── Event forwarding ──────────────────────────────────

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
					ToolName:  toolUseID,
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

// ─── Helpers ───────────────────────────────────────────

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
