import React, { useEffect, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
  Alert,
} from 'react-native';
import { useTranslation } from 'react-i18next';
import { useAgentStore } from '../stores/agentStore';
import { useTheme } from '../hooks/useTheme';
import { Brand } from '../theme/colors';

const AGENT_COLORS: Record<string, string> = {
  cyan: '#06B6D4',
  purple: '#A855F7',
  blue: '#3B82F6',
  green: '#22C55E',
  orange: '#F97316',
  pink: '#EC4899',
  red: '#EF4444',
  yellow: '#EAB308',
};

export default function AgentListScreen() {
  const { agents, loading, fetchAgents, deleteAgent } = useAgentStore();
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  const handleRefresh = useCallback(() => {
    fetchAgents();
  }, [fetchAgents]);

  const handleDelete = (name: string, source: string) => {
    Alert.alert(
      t('agent.delete'),
      t('agent.deleteConfirm', { name }),
      [
        { text: t('session.cancel'), style: 'cancel' },
        {
          text: t('common.delete'),
          style: 'destructive',
          onPress: () => deleteAgent(name),
        },
      ]
    );
  };

  const renderItem = ({ item }: { item: typeof agents[0] }) => {
    const avatarColor = AGENT_COLORS[item.color || ''] || Brand.primary;
    const initial = (item.name || '?')[0].toUpperCase();

    return (
      <TouchableOpacity
        style={[styles.agentCard, { backgroundColor: c.surface, borderColor: c.surfaceBorder }]}
        onLongPress={() => handleDelete(item.name, item.source)}
        activeOpacity={0.7}
      >
        <View style={[styles.avatar, { backgroundColor: avatarColor }]}>
          <Text style={styles.avatarText}>{initial}</Text>
        </View>
        <View style={styles.agentInfo}>
          <View style={styles.nameRow}>
            <Text style={[styles.agentName, { color: c.textPrimary }]}>{item.name}</Text>
            {item.source === 'project' && (
              <View style={[styles.projectBadge, { backgroundColor: c.toolRead + '20' }]}>
                <Text style={[styles.projectBadgeText, { color: c.toolRead }]}>{t('agent.project')}</Text>
              </View>
            )}
          </View>
          <Text style={[styles.agentDesc, { color: c.textSecondary }]} numberOfLines={2}>
            {item.description || t('agent.noDesc')}
          </Text>
          {item.model && (
            <Text style={[styles.agentModel, { color: c.textTertiary }]}>
              {t('agent.model', { model: item.model })}
            </Text>
          )}
        </View>
      </TouchableOpacity>
    );
  };

  return (
    <View style={[styles.container, { backgroundColor: c.background }]}>
      {agents.length === 0 && !loading ? (
        <View style={styles.emptyState}>
          <Text style={[styles.emptyTitle, { color: c.textPrimary }]}>{t('agent.empty')}</Text>
          <Text style={[styles.emptyHint, { color: c.textTertiary }]}>
            {t('agent.emptyHint')}
          </Text>
        </View>
      ) : (
        <FlatList
          data={agents}
          renderItem={renderItem}
          keyExtractor={(item) => `${item.source}-${item.name}`}
          contentContainerStyle={styles.listContent}
          refreshControl={
            <RefreshControl refreshing={loading} onRefresh={handleRefresh} />
          }
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  listContent: {
    padding: 16,
  },
  agentCard: {
    flexDirection: 'row',
    borderRadius: 16,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
  },
  avatar: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 14,
  },
  avatarText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '700',
  },
  agentInfo: {
    flex: 1,
  },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 4,
  },
  agentName: {
    fontSize: 16,
    fontWeight: '600',
  },
  projectBadge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 6,
  },
  projectBadgeText: {
    fontSize: 11,
    fontWeight: '500',
  },
  agentDesc: {
    fontSize: 14,
    lineHeight: 20,
  },
  agentModel: {
    fontSize: 12,
    marginTop: 4,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 40,
  },
  emptyTitle: {
    fontSize: 20,
    fontWeight: '700',
    marginBottom: 8,
  },
  emptyHint: {
    fontSize: 14,
    textAlign: 'center',
    lineHeight: 22,
  },
});
