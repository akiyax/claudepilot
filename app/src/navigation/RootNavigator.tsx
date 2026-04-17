import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { useConnectionStore } from '../stores/connectionStore';
import { useChatStore } from '../stores/chatStore';
import ConnectionScreen from '../screens/ConnectionScreen';
import MainTabs from './MainTabs';
import type { WSMessage } from '../types/ws';

export type RootStackParamList = {
  Connection: undefined;
  Main: undefined;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function RootNavigator() {
  const { connection, loadSavedConnection } = useConnectionStore();
  const { handleWSMessage } = useChatStore();

  // Route WS messages to appropriate stores
  useEffect(() => {
    const { handleMessage: handleConnectionMsg } = useConnectionStore.getState();

    // Set up unified WS message handler
    const originalOnMessage = (msg: WSMessage) => {
      handleConnectionMsg(msg);
      handleWSMessage(msg);
    };

    // Replace the WS handler with our unified one
    const ws = require('../services/WebSocketService').wsService;
    ws.onMessage(originalOnMessage);

    return () => {
      ws.onMessage(() => {});
    };
  }, [handleWSMessage]);

  // Load saved connection on mount
  useEffect(() => {
    loadSavedConnection();
  }, [loadSavedConnection]);

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
