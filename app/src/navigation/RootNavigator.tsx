import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { useConnectionStore } from '../stores/connectionStore';
import { useChatStore } from '../stores/chatStore';
import { useAgentStore } from '../stores/agentStore';
import { useSessionStore } from '../stores/sessionStore';
import { wsService } from '../services/WebSocketService';
import ConnectionScreen from '../screens/ConnectionScreen';
import MainTabs from './MainTabs';
import type { WSMessage } from '../types/ws';
import type { AgentItem, SessionItem, HistoryMessage } from '../types/ws';

export type RootStackParamList = {
  Connection: undefined;
  Main: undefined;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

// Unified WS message dispatcher — routes to all stores
function dispatchWSMessage(msg: WSMessage) {
  // Connection store
  useConnectionStore.getState().handleMessage(msg);

  // Chat store
  useChatStore.getState().handleWSMessage(msg);

  // Agent store
  if (msg.type === 'agent.list.result' && msg.payload) {
    useAgentStore.getState().handleAgentListResult(msg.payload.agents as AgentItem[]);
  }

  // Session store
  if (msg.type === 'session.list.result' && msg.payload) {
    useSessionStore.getState().handleSessionListResult(msg.payload.sessions as SessionItem[]);
  }
  if (msg.type === 'session.history.result' && msg.payload) {
    useSessionStore.getState().handleSessionHistoryResult(msg.payload.messages as HistoryMessage[]);
  }
  if (msg.type === 'session.updated') {
    useSessionStore.getState().fetchSessions();
  }
}

export default function RootNavigator() {
  const { connection, loadSavedConnection } = useConnectionStore();

  // Set up WS message routing
  useEffect(() => {
    wsService.onMessage(dispatchWSMessage);
    return () => { wsService.onMessage(() => {}); };
  }, []);

  // Auto-reconnect on mount
  useEffect(() => {
    loadSavedConnection();
  }, [loadSavedConnection]);

  // Fetch initial data when connected
  useEffect(() => {
    if (connection.connected) {
      useAgentStore.getState().fetchAgents();
      useSessionStore.getState().fetchSessions();
    }
  }, [connection.connected]);

  return (
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {connection.connected ? (
          <Stack.Screen name="Main" component={MainTabs} />
        ) : (
          <Stack.Screen name="Connection" component={ConnectionScreen} />
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
