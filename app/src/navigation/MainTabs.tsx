import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import ChatScreen from '../screens/ChatScreen';
import SessionListScreen from '../screens/SessionListScreen';
import AgentListScreen from '../screens/AgentListScreen';
import SettingsScreen from '../screens/SettingsScreen';

export type MainTabParamList = {
  Chat: undefined;
  Sessions: undefined;
  Agents: undefined;
  Settings: undefined;
};

const Tab = createBottomTabNavigator<MainTabParamList>();

export default function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: true,
        tabBarLabelStyle: { fontSize: 12 },
      }}
    >
      <Tab.Screen
        name="Chat"
        component={ChatScreen}
        options={{ title: '对话', tabBarLabel: '对话' }}
      />
      <Tab.Screen
        name="Sessions"
        component={SessionListScreen}
        options={{ title: '会话', tabBarLabel: '会话' }}
      />
      <Tab.Screen
        name="Agents"
        component={AgentListScreen}
        options={{ title: 'Agent', tabBarLabel: 'Agent' }}
      />
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: '设置', tabBarLabel: '设置' }}
      />
    </Tab.Navigator>
  );
}
