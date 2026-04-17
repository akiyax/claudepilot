import { create } from 'zustand';
import { wsService } from '../services/WebSocketService';
import type { SessionItem, HistoryMessage } from '../types/ws';

interface SessionState {
  sessions: SessionItem[];
  history: HistoryMessage[];
  loading: boolean;
  error: string | null;

  fetchSessions: (projectDir?: string) => void;
  fetchHistory: (sessionId: string, limit?: number) => void;
  deleteSession: (sessionId: string) => void;
  resumeSession: (sessionId: string) => void;

  handleSessionListResult: (sessions: SessionItem[]) => void;
  handleSessionHistoryResult: (messages: HistoryMessage[]) => void;
}

export const useSessionStore = create<SessionState>((set, get) => ({
  sessions: [],
  history: [],
  loading: false,
  error: null,

  fetchSessions: (projectDir?: string) => {
    set({ loading: true, error: null });
    wsService.send('session.list', { projectDir });
  },

  fetchHistory: (sessionId: string, limit: number = 50) => {
    set({ loading: true, history: [] });
    wsService.send('session.history', { sessionId, limit });
  },

  deleteSession: (sessionId: string) => {
    wsService.send('session.delete', { sessionId });
    setTimeout(() => get().fetchSessions(), 500);
  },

  resumeSession: (sessionId: string) => {
    wsService.send('session.resume', { sessionId });
  },

  handleSessionListResult: (sessions: SessionItem[]) => {
    set({ sessions, loading: false });
  },

  handleSessionHistoryResult: (messages: HistoryMessage[]) => {
    set({ history: messages, loading: false });
  },
}));
