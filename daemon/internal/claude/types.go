package claude

import "time"

// ─── Claude Code stdout event types ───────────────────────
// These are parsed from Claude Code CLI's --output-format stream-json output.
// Each line is a complete JSON object (NDJSON format).

// ClaudeEvent is a raw event read from Claude Code's stdout.
// The "type" field determines how to interpret the rest of the fields.
type ClaudeEvent struct {
	Type    string         `json:"type"`
	Raw     map[string]any `json:"-"` // preserved for unknown event types
}

// SystemInit is the first event emitted when Claude Code starts a session.
// type: "system", subtype: "init"
type SystemInit struct {
	Type             string   `json:"type"`
	Subtype          string   `json:"subtype"`
	SessionID        string   `json:"session_id"`
	ClaudeCodeVersion string  `json:"claude_code_version"`
	Cwd              string   `json:"cwd"`
	Model            string   `json:"model"`
	Tools            []string `json:"tools"`
	McpServers       []any    `json:"mcp_servers"`
	ApiKeySource     string   `json:"apiKeySource"`
	UUID             string   `json:"uuid"`
}

// AssistantMessage represents an assistant turn with content blocks.
// type: "assistant"
type AssistantMessage struct {
	Type            string         `json:"type"`
	SessionID       string         `json:"session_id"`
	UUID            string         `json:"uuid"`
	Message         AssistantMsg   `json:"message"`
	ParentToolUseID *string        `json:"parent_tool_use_id"`
}

// AssistantMsg holds the nested message object.
type AssistantMsg struct {
	Role      string         `json:"role"`
	Content   []ContentBlock `json:"content"`
	StopReason string        `json:"stop_reason,omitempty"`
	Model     string         `json:"model,omitempty"`
	Usage     *TokenUsage    `json:"usage,omitempty"`
}

// ContentBlock represents a single content block within an assistant message.
type ContentBlock struct {
	Type  string `json:"type"` // "text", "tool_use", "thinking"

	// For type="text"
	Text string `json:"text,omitempty"`

	// For type="tool_use"
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Input any   `json:"input,omitempty"`

	// For type="thinking"
	Thinking string `json:"thinking,omitempty"`
}

// TokenUsage represents token consumption statistics.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheReadInputTokens    int `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens,omitempty"`
}

// ResultMessage is the final message of a turn.
// type: "result"
type ResultMessage struct {
	Type             string          `json:"type"`
	Subtype          string          `json:"subtype"` // "success" or "error"
	SessionID        string          `json:"session_id"`
	UUID             string          `json:"uuid"`
	Content          string          `json:"content"`
	Usage            *ResultUsage    `json:"usage,omitempty"`
	ModelUsage       map[string]ModelUsageEntry `json:"modelUsage,omitempty"`
	PermissionDenials []any          `json:"permission_denials,omitempty"`
	Errors           []string        `json:"errors,omitempty"`
}

// ResultUsage contains detailed usage statistics at the end of a turn.
type ResultUsage struct {
	InputTokens     int     `json:"inputTokens"`
	OutputTokens    int     `json:"outputTokens"`
	ContextWindow   int     `json:"contextWindow,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	CostUSD         float64 `json:"costUSD,omitempty"`
	CacheReadInputTokens    int `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens,omitempty"`
}

// ModelUsageEntry tracks per-model usage.
type ModelUsageEntry struct {
	InputTokens     int     `json:"inputTokens"`
	OutputTokens    int     `json:"outputTokens"`
	CostUSD         float64 `json:"costUSD,omitempty"`
}

// ─── Permission protocol ─────────────────────────────────

// ControlRequest is a permission request from Claude Code.
// type: "control_request"
type ControlRequest struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Request   PermissionInput `json:"request"`
}

// PermissionInput describes what Claude Code wants to do.
type PermissionInput struct {
	Subtype              string           `json:"subtype"` // "can_use_tool"
	ToolName             string           `json:"tool_name"`
	Input                any              `json:"input"`
	PermissionSuggestions []PermissionSuggestion `json:"permission_suggestions,omitempty"`
	Title                string           `json:"title,omitempty"`
	DisplayName          string           `json:"display_name,omitempty"`
	ToolUseID            string           `json:"tool_use_id,omitempty"`
	Description          string           `json:"description,omitempty"`
}

// PermissionSuggestion is a suggested permission rule.
type PermissionSuggestion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Allowed     bool   `json:"allowed"`
}

// ─── stdin input types ───────────────────────────────────
// These are written to Claude Code's stdin.

// UserInput is sent to stdin to deliver a user message.
type UserInput struct {
	Type    string      `json:"type"`
	Message UserMessage `json:"message"`
}

// UserMessage holds the user's message content.
type UserMessage struct {
	Role    string `json:"role"` // "user"
	Content any    `json:"content"` // string or []ContentBlock for attachments
}

// ControlResponse is sent to stdin to respond to a permission request.
type ControlResponse struct {
	Type     string              `json:"type"`
	Response ControlResponseBody `json:"response"`
}

// ControlResponseBody wraps the actual permission decision.
type ControlResponseBody struct {
	Subtype   string      `json:"subtype"` // "success"
	RequestID string      `json:"request_id"`
	Response  any         `json:"response"` // PermissionDecision
}

// PermissionAllow is the response when allowing a tool.
type PermissionAllow struct {
	Behavior     string         `json:"behavior"` // "allow"
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
}

// PermissionDeny is the response when denying a tool.
type PermissionDeny struct {
	Behavior string `json:"behavior"` // "deny"
	Message  string `json:"message,omitempty"`
}

// ─── AskUserQuestion types ────────────────────────────────

// AskUserQuestionInput represents the tool_use input for ask_user_question.
// This appears as a tool_use content block with name="ask_user_question".
type AskUserQuestionInput struct {
	Questions []Question `json:"questions"`
}

// Question represents a single question with options.
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header,omitempty"`
	Options     []Option `json:"options,omitempty"`
	MultiSelect bool     `json:"multiSelect,omitempty"`
}

// Option represents a selectable option in a question.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Preview     string `json:"preview,omitempty"`
}

// ─── WS Message types (Daemon ↔ APP) ─────────────────────
// These are the WebSocket message types for the APP communication layer.

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type      string         `json:"type"`
	ID        string         `json:"id,omitempty"`
	Timestamp int64          `json:"timestamp"`
	DaemonID  string         `json:"daemonId,omitempty"`
	Payload   any            `json:"payload,omitempty"`
}

// NewWSMessage creates a new WS message with current timestamp.
func NewWSMessage(msgType string, payload any) WSMessage {
	return WSMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

// ─── WS Payload types ────────────────────────────────────

// SystemHelloPayload is sent when a client connects.
type SystemHelloPayload struct {
	Version      string   `json:"version"`
	DaemonID     string   `json:"daemonId"`
	CLIVersion   string   `json:"cliVersion"`
	Mode         string   `json:"mode"` // "lan"
	Capabilities []string `json:"capabilities"`
	Commands     []string `json:"commands"`
}

// SystemStatusPayload is sent periodically.
type SystemStatusPayload struct {
	Connected      bool   `json:"connected"`
	DaemonVersion  string `json:"daemonVersion"`
	CLIVersion     string `json:"cliVersion"`
	Uptime         int64  `json:"uptime"`
	UpdateAvailable *string `json:"updateAvailable,omitempty"`
}

// SystemAckPayload confirms receipt of a critical message.
type SystemAckPayload struct {
	RefID    string `json:"refId"`
	Received bool   `json:"received"`
}

// StreamTextPayload carries streaming text.
type StreamTextPayload struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

// StreamThinkingPayload carries thinking content.
type StreamThinkingPayload struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

// StreamEndPayload signals the end of an assistant turn.
type StreamEndPayload struct {
	SessionID      string       `json:"sessionId"`
	Usage          StreamUsage  `json:"usage"`
}

// StreamUsage carries token usage at the end of a stream.
type StreamUsage struct {
	InputTokens   int `json:"inputTokens"`
	OutputTokens  int `json:"outputTokens"`
	TotalTokens   int `json:"totalTokens"`
	ContextWindow int `json:"contextWindow"`
	UsedPercent   int `json:"usedPercent"`
}

// SessionTitlePayload carries an AI-generated session title.
type SessionTitlePayload struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
}

// ToolCallPayload signals a tool invocation start.
type ToolCallPayload struct {
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	ToolInput any    `json:"toolInput"`
	ToolID    string `json:"toolID,omitempty"`
}

// ToolOutputPayload carries tool execution output.
type ToolOutputPayload struct {
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	Result    string `json:"result"`
	IsError   bool   `json:"isError"`
}

// ToolEndPayload signals a tool invocation completed.
type ToolEndPayload struct {
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	Success   bool   `json:"success"`
}

// PermissionRequestPayload is sent to APP for user approval.
type PermissionRequestPayload struct {
	RequestID  string `json:"requestId"`
	ToolName   string `json:"toolName"`
	ToolInput  any    `json:"toolInput"`
	ToolUseID  string `json:"toolUseId,omitempty"`
	Title      string `json:"title,omitempty"`
	DisplayText string `json:"displayText,omitempty"`
}

// PermissionRespondPayload is sent from APP to approve/deny.
type PermissionRespondPayload struct {
	RequestID    string         `json:"requestId"`
	Behavior     string         `json:"behavior"` // "allow" or "deny"
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	Message      string         `json:"message,omitempty"`
}

// QuestionAskPayload is sent to APP when Claude asks a question.
type QuestionAskPayload struct {
	RequestID string     `json:"requestId"`
	Questions []Question `json:"questions"`
	ToolUseID string     `json:"toolUseId,omitempty"`
}

// QuestionAnswerPayload is sent from APP with user's answers.
type QuestionAnswerPayload struct {
	RequestID string         `json:"requestId"`
	Answers   map[string]string `json:"answers"`
}

// AgentAttachmentPayload carries a file generated by the agent.
type AgentAttachmentPayload struct {
	SessionID string `json:"sessionId"`
	FileName  string `json:"fileName"`
	FileType  string `json:"fileType"` // image/video/pdf/audio/markdown/code/other
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	Data      string `json:"data,omitempty"` // base64 for ≤2MB
	URL       string `json:"url,omitempty"`  // download URL for >2MB
	Preview   string `json:"preview,omitempty"`
}

// TaskPayload carries task lifecycle events.
type TaskPayload struct {
	ID         string `json:"id"`
	Subject    string `json:"subject"`
	ActiveForm string `json:"activeForm,omitempty"`
	Status     string `json:"status,omitempty"` // pending/in_progress/completed
}

// AgentSubPayload carries sub-agent lifecycle events.
type AgentSubPayload struct {
	AgentID     string `json:"agentId"`
	Type        string `json:"type"` // Explore, Plan, etc.
	Description string `json:"description"`
	Progress    string `json:"progress,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// ErrorPayload carries error messages.
type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ─── APP request payloads ────────────────────────────────

// ChatMessagePayload is sent from APP with user text.
type ChatMessagePayload struct {
	Text   string   `json:"text"`
	Images []string `json:"images,omitempty"` // base64
	Files  []string `json:"files,omitempty"`  // base64
}

// ChatInterruptPayload is sent to interrupt or queue a message.
type ChatInterruptPayload struct {
	Text string `json:"text"`
	Mode string `json:"mode"` // "interrupt" or "queue"
}

// SessionStartPayload initiates a new Claude Code session.
type SessionStartPayload struct {
	ProjectDir     string `json:"projectDir,omitempty"`
	AgentName      string `json:"agentName,omitempty"`
	Model          string `json:"model,omitempty"`
	Provider       string `json:"provider,omitempty"`
	PermissionMode string `json:"permissionMode,omitempty"`
	Effort         string `json:"effort,omitempty"`
	SessionName    string `json:"sessionName,omitempty"`
}

// SessionResumePayload resumes an existing session.
type SessionResumePayload struct {
	SessionID string `json:"sessionId"`
}

// SessionDeletePayload deletes a session.
type SessionDeletePayload struct {
	SessionID string `json:"sessionId"`
}

// SessionHistoryPayload requests session history.
type SessionHistoryPayload struct {
	SessionID string `json:"sessionId"`
	Limit     int    `json:"limit,omitempty"`
}

// SlashCommandPayload executes a slash command.
type SlashCommandPayload struct {
	Command string `json:"command"`
}

// ModelSwitchPayload switches the active model.
type ModelSwitchPayload struct {
	Model string `json:"model"`
}

// EffortSwitchPayload switches the reasoning effort level.
type EffortSwitchPayload struct {
	Effort string `json:"effort"`
}

// ModeSwitchPayload switches the permission mode.
type ModeSwitchPayload struct {
	Mode string `json:"mode"`
}

// PlanApprovePayload approves or rejects a plan.
type PlanApprovePayload struct {
	Approved bool   `json:"approved"`
	PlanID   string `json:"planId,omitempty"`
}
