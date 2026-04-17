// WS message types for Daemon ↔ APP communication

export interface WSMessage {
  type: string;
  id?: string;
  timestamp: number;
  daemonId?: string;
  payload?: any;
}

// ─── System ────────────────────────────────────

export interface SystemHelloPayload {
  version: string;
  daemonId: string;
  cliVersion: string;
  mode: string;
  capabilities: string[];
  commands: string[];
}

export interface SystemAckPayload {
  refId: string;
  received: boolean;
}

// ─── Chat ──────────────────────────────────────

export interface ChatMessagePayload {
  text: string;
  images?: string[];
  files?: string[];
}

export interface StreamTextPayload {
  sessionId: string;
  content: string;
}

export interface StreamThinkingPayload {
  sessionId: string;
  content: string;
}

export interface StreamEndPayload {
  sessionId: string;
  usage: StreamUsage;
}

export interface StreamUsage {
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  contextWindow: number;
  usedPercent: number;
}

// ─── Tools ─────────────────────────────────────

export interface ToolCallPayload {
  sessionId: string;
  toolName: string;
  toolInput: any;
  toolID?: string;
}

export interface ToolOutputPayload {
  sessionId: string;
  toolName: string;
  result: string;
  isError: boolean;
}

// ─── Permission ────────────────────────────────

export interface PermissionRequestPayload {
  requestId: string;
  toolName: string;
  toolInput: any;
  toolUseId?: string;
  title?: string;
  displayText?: string;
}

export interface PermissionRespondPayload {
  requestId: string;
  behavior: 'allow' | 'deny';
  updatedInput?: Record<string, any>;
  message?: string;
}

// ─── Session ───────────────────────────────────

export interface SessionStartPayload {
  projectDir?: string;
  agentName?: string;
  model?: string;
  provider?: string;
  permissionMode?: string;
  effort?: string;
  sessionName?: string;
}

export interface SessionResumePayload {
  sessionId: string;
}

export interface SessionItem {
  id: string;
  projectDir?: string;
  summary: string;
  messageCount: number;
  modifiedAt: number;
}

export interface HistoryMessage {
  type: 'user' | 'assistant';
  content: string;
  timestamp: number;
}

// ─── Agent ─────────────────────────────────────

export interface AgentItem {
  name: string;
  description: string;
  model?: string;
  color?: string;
  source: 'user' | 'project';
}

export interface AgentCreatePayload {
  name: string;
  description?: string;
  prompt: string;
  model?: string;
  tools?: string[];
  disallowedTools?: string[];
  permissionMode?: string;
  effort?: string;
  maxTurns?: number;
  memory?: string;
  color?: string;
  initialPrompt?: string;
  isolation?: string;
  background?: boolean;
  projectDir?: string;
}

// ─── Provider ──────────────────────────────────

export interface ProviderItem {
  name: string;
  isDefault: boolean;
  baseUrl?: string;
  model?: string;
}

// ─── Error ─────────────────────────────────────

export interface ErrorPayload {
  message: string;
  code?: string;
}
