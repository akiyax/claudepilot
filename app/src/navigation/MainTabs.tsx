import React from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation();

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
        options={{ title: t('tabs.chat'), tabBarLabel: t('tabs.chat') }}
      />
      <Tab.Screen
        name="Sessions"
        component={SessionListScreen}
        options={{ title: t('tabs.sessions'), tabBarLabel: t('tabs.sessions') }}
      />
      <Tab.Screen
        name="Agents"
        component={AgentListScreen}
        options={{ title: t('tabs.agents'), tabBarLabel: t('tabs.agents') }}
      />
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: t('tabs.settings'), tabBarLabel: t('tabs.settings') }}
      />
    </Tab.Navigator>
  );
}
