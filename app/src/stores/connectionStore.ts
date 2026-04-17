import { create } from 'zustand';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { wsService } from '../services/WebSocketService';
import type { WSMessage, SystemHelloPayload } from '../types/ws';
import type { ConnectionInfo } from '../types/models';

const STORAGE_KEY = '@claudepilot_connection';

interface ConnectionState {
  connection: ConnectionInfo;
  isConnecting: boolean;
  error: string | null;
  daemonVersion: string | null;
  cliVersion: string | null;

  // Actions
  connectByQR: (url: string, token: string) => Promise<void>;
  connectByPairingCode: (host: string, port: number, code: string) => Promise<void>;
  disconnect: () => void;
  loadSavedConnection: () => Promise<void>;
  handleMessage: (msg: WSMessage) => void;
}

export const useConnectionStore = create<ConnectionState>((set, get) => ({
  connection: {
    url: '',
    connected: false,
  },
  isConnecting: false,
  error: null,
  daemonVersion: null,
  cliVersion: null,

  connectByQR: async (url: string, token: string) => {
    set({ isConnecting: true, error: null });
    try {
      await wsService.connect(url, token);
      wsService.onMessage(get().handleMessage);
      // Send ready after connection
      wsService.send('system.ready');

      // Save connection info
      await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify({ url, token }));

      set({
        connection: { url, token, connected: true },
        isConnecting: false,
      });
    } catch (err: any) {
      set({
        isConnecting: false,
        error: err.message || 'Connection failed',
      });
      throw err;
    }
  },

  connectByPairingCode: async (host: string, port: number, code: string) => {
    set({ isConnecting: true, error: null });
    try {
      // Step 1: Validate pairing code via HTTP
      const response = await fetch(`http://${host}:${port}/pair?code=${code}`);
      if (!response.ok) {
        throw new Error('Invalid pairing code');
      }
      const data = await response.json();
      const token = data.token;

      // Step 2: Connect via WS
      const wsUrl = `ws://${host}:${port}/ws`;
      await wsService.connect(wsUrl, token);
      wsService.onMessage(get().handleMessage);
      wsService.send('system.ready');

      await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify({ url: wsUrl, token }));

      set({
        connection: { url: wsUrl, token, connected: true },
        isConnecting: false,
      });
    } catch (err: any) {
      set({
        isConnecting: false,
        error: err.message || 'Pairing failed',
      });
      throw err;
    }
  },

  disconnect: () => {
    wsService.disconnect();
    AsyncStorage.removeItem(STORAGE_KEY);
    set({
      connection: { url: '', connected: false },
      daemonVersion: null,
      cliVersion: null,
    });
  },

  loadSavedConnection: async () => {
    try {
      const saved = await AsyncStorage.getItem(STORAGE_KEY);
      if (saved) {
        const { url, token } = JSON.parse(saved);
        set({ isConnecting: true });
        try {
          await wsService.connect(url, token);
          wsService.onMessage(get().handleMessage);
          wsService.send('system.ready');
          set({
            connection: { url, token, connected: true },
            isConnecting: false,
          });
        } catch {
          // Saved connection failed, clear it
          AsyncStorage.removeItem(STORAGE_KEY);
          set({ isConnecting: false });
        }
      }
    } catch {
      // Ignore storage errors
    }
  },

  handleMessage: (msg: WSMessage) => {
    switch (msg.type) {
      case 'system.hello': {
        const payload = msg.payload as SystemHelloPayload;
        set({
          daemonVersion: payload.version,
          cliVersion: payload.cliVersion,
          connection: {
            ...get().connection,
            daemonId: payload.daemonId,
            daemonVersion: payload.version,
            cliVersion: payload.cliVersion,
            connected: true,
          },
        });
        break;
      }
      case 'error': {
        set({ error: msg.payload?.message || 'Unknown error' });
        break;
      }
    }
  },
}));
