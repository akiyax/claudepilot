import { create } from 'zustand';
import { wsService } from '../services/WebSocketService';
import type { AgentItem, AgentCreatePayload } from '../types/ws';

interface AgentState {
  agents: AgentItem[];
  loading: boolean;
  error: string | null;

  fetchAgents: (projectDir?: string) => void;
  createAgent: (payload: AgentCreatePayload) => void;
  deleteAgent: (name: string, projectDir?: string) => void;

  handleAgentListResult: (agents: AgentItem[]) => void;
}

export const useAgentStore = create<AgentState>((set, get) => ({
  agents: [],
  loading: false,
  error: null,

  fetchAgents: (projectDir?: string) => {
    set({ loading: true, error: null });
    wsService.send('agent.list', { projectDir });
  },

  createAgent: (payload: AgentCreatePayload) => {
    wsService.send('agent.create', payload);
    // Refresh list after creation
    setTimeout(() => get().fetchAgents(payload.projectDir), 500);
  },

  deleteAgent: (name: string, projectDir?: string) => {
    wsService.send('agent.delete', { name, projectDir });
    // Refresh list after deletion
    setTimeout(() => get().fetchAgents(projectDir), 500);
  },

  handleAgentListResult: (agents: AgentItem[]) => {
    set({ agents, loading: false });
  },
}));
