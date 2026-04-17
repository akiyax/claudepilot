// Domain models for the APP

export interface ChatMessage {
  id: string;
  type: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: number;
  sessionId?: string;
  // For assistant messages
  isStreaming?: boolean;
  thinkingContent?: string;
  // For tool calls
  toolCalls?: ToolCall[];
}

export interface ToolCall {
  id: string;
  name: string;
  input?: any;
  output?: string;
  isError?: boolean;
  status: 'running' | 'completed' | 'failed';
}

export interface PermissionRequest {
  requestId: string;
  toolName: string;
  toolInput: any;
  title?: string;
  displayText?: string;
}

export interface QuestionRequest {
  requestId: string;
  questions: Question[];
}

export interface Question {
  question: string;
  header?: string;
  options?: QuestionOption[];
  multiSelect?: boolean;
}

export interface QuestionOption {
  label: string;
  description?: string;
  preview?: string;
}

export interface ConnectionInfo {
  url: string;
  token?: string;
  daemonId?: string;
  connected: boolean;
  daemonVersion?: string;
  cliVersion?: string;
}

export interface SessionConfig {
  projectDir?: string;
  agentName?: string;
  model?: string;
  provider?: string;
  permissionMode?: string;
  effort?: string;
}

export interface ContextUsage {
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  contextWindow: number;
  usedPercent: number;
}
