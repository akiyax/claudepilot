import { create } from 'zustand';
import { wsService } from '../services/WebSocketService';
import type { WSMessage, StreamTextPayload, StreamThinkingPayload, StreamEndPayload, PermissionRequestPayload, ToolCallPayload, ToolOutputPayload } from '../types/ws';
import type { ChatMessage, ToolCall, PermissionRequest, QuestionRequest, ContextUsage, SessionConfig } from '../types/models';

interface ChatState {
  messages: ChatMessage[];
  isStreaming: boolean;
  currentAssistantText: string;
  currentThinkingText: string;
  activeToolCalls: Map<string, ToolCall>;
  permissionRequest: PermissionRequest | null;
  questionRequest: QuestionRequest | null;
  sessionTitle: string;
  contextUsage: ContextUsage;
  currentSession: SessionConfig | null;
  sessionId: string | null;

  // Actions
  sendMessage: (text: string) => void;
  respondPermission: (requestId: string, behavior: 'allow' | 'deny', updatedInput?: Record<string, any>) => void;
  handleWSMessage: (msg: WSMessage) => void;
  startSession: (config: SessionConfig) => void;
  clearMessages: () => void;
}

let messageCounter = 0;
function nextId(): string {
  return `msg-${Date.now()}-${++messageCounter}`;
}

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  isStreaming: false,
  currentAssistantText: '',
  currentThinkingText: '',
  activeToolCalls: new Map(),
  permissionRequest: null,
  questionRequest: null,
  sessionTitle: '新对话',
  contextUsage: { inputTokens: 0, outputTokens: 0, totalTokens: 0, contextWindow: 0, usedPercent: 0 },
  currentSession: null,
  sessionId: null,

  sendMessage: (text: string) => {
    // Add user message to local state
    const userMsg: ChatMessage = {
      id: nextId(),
      type: 'user',
      content: text,
      timestamp: Date.now(),
    };
    set(state => ({ messages: [...state.messages, userMsg] }));

    // Send to daemon
    wsService.send('chat.message', { text });
  },

  respondPermission: (requestId: string, behavior: 'allow' | 'deny', updatedInput?: Record<string, any>) => {
    wsService.send('permission.respond', {
      requestId,
      behavior,
      updatedInput,
    });
    set({ permissionRequest: null });
  },

  handleWSMessage: (msg: WSMessage) => {
    const state = get();

    switch (msg.type) {
      case 'stream.text': {
        const payload = msg.payload as StreamTextPayload;
        if (!state.isStreaming) {
          // Start a new assistant message
          const assistantMsg: ChatMessage = {
            id: nextId(),
            type: 'assistant',
            content: payload.content,
            timestamp: Date.now(),
            isStreaming: true,
          };
          set({
            messages: [...state.messages, assistantMsg],
            isStreaming: true,
            currentAssistantText: payload.content,
          });
        } else {
          // Append to current assistant message
          const newMessages = [...state.messages];
          const lastMsg = newMessages[newMessages.length - 1];
          if (lastMsg && lastMsg.type === 'assistant') {
            lastMsg.content = state.currentAssistantText + payload.content;
          }
          set({
            messages: newMessages,
            currentAssistantText: state.currentAssistantText + payload.content,
          });
        }
        break;
      }

      case 'stream.thinking': {
        const payload = msg.payload as StreamThinkingPayload;
        set({
          currentThinkingText: state.currentThinkingText + payload.content,
        });
        break;
      }

      case 'stream.end': {
        const payload = msg.payload as StreamEndPayload;
        const newMessages = [...state.messages];
        const lastMsg = newMessages[newMessages.length - 1];
        if (lastMsg && lastMsg.type === 'assistant' && lastMsg.isStreaming) {
          lastMsg.isStreaming = false;
        }
        set({
          messages: newMessages,
          isStreaming: false,
          currentAssistantText: '',
          currentThinkingText: '',
          sessionId: payload.sessionId,
          contextUsage: payload.usage,
        });
        break;
      }

      case 'tool.call': {
        const payload = msg.payload as ToolCallPayload;
        const toolCall: ToolCall = {
          id: payload.toolID || nextId(),
          name: payload.toolName,
          input: payload.toolInput,
          status: 'running',
        };
        const newToolCalls = new Map(state.activeToolCalls);
        newToolCalls.set(toolCall.id, toolCall);
        set({ activeToolCalls: newToolCalls });
        break;
      }

      case 'tool.output': {
        const payload = msg.payload as ToolOutputPayload;
        const newToolCalls = new Map(state.activeToolCalls);
        // Find the tool call by name (toolName is actually toolUseID in some cases)
        for (const [id, tc] of newToolCalls) {
          if (tc.name === payload.toolName || id === payload.toolName) {
            tc.output = payload.result;
            tc.isError = payload.isError;
            tc.status = payload.isError ? 'failed' : 'completed';
            break;
          }
        }
        set({ activeToolCalls: newToolCalls });
        break;
      }

      case 'permission.request': {
        const payload = msg.payload as PermissionRequestPayload;
        set({
          permissionRequest: {
            requestId: payload.requestId,
            toolName: payload.toolName,
            toolInput: payload.toolInput,
            title: payload.title,
            displayText: payload.displayText,
          },
        });
        break;
      }

      case 'session.title': {
        set({ sessionTitle: msg.payload?.title || '新对话' });
        break;
      }

      case 'error': {
        console.error('[ChatStore] Error:', msg.payload?.message);
        break;
      }
    }
  },

  startSession: (config: SessionConfig) => {
    set({
      messages: [],
      isStreaming: false,
      currentAssistantText: '',
      currentThinkingText: '',
      activeToolCalls: new Map(),
      permissionRequest: null,
      questionRequest: null,
      sessionTitle: '新对话',
      currentSession: config,
    });
    wsService.send('session.start', config);
  },

  clearMessages: () => {
    set({
      messages: [],
      isStreaming: false,
      currentAssistantText: '',
      currentThinkingText: '',
      activeToolCalls: new Map(),
    });
  },
}));
