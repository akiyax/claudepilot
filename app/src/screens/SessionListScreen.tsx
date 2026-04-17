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
import { useSessionStore } from '../stores/sessionStore';
import { useChatStore } from '../stores/chatStore';
import { useTheme } from '../hooks/useTheme';

export default function SessionListScreen() {
  const { sessions, loading, fetchSessions, deleteSession, resumeSession } = useSessionStore();
  const { startSession } = useChatStore();
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  const handleRefresh = useCallback(() => {
    fetchSessions();
  }, [fetchSessions]);

  const handleResume = (sessionId: string) => {
    resumeSession(sessionId);
    startSession({}); // Reset chat state
  };

  const handleDelete = (sessionId: string, summary: string) => {
    Alert.alert(
      t('session.delete'),
      t('session.deleteConfirm', { name: summary || sessionId.slice(0, 8) }),
      [
        { text: t('session.cancel'), style: 'cancel' },
        {
          text: t('common.delete'),
          style: 'destructive',
          onPress: () => deleteSession(sessionId),
        },
      ]
    );
  };

  const renderItem = ({ item }: { item: typeof sessions[0] }) => {
    return (
      <TouchableOpacity
        style={[styles.sessionCard, { backgroundColor: c.surface, borderColor: c.surfaceBorder }]}
        onPress={() => handleResume(item.id)}
        onLongPress={() => handleDelete(item.id, item.summary)}
        activeOpacity={0.7}
      >
        <View style={[styles.sessionIcon, { backgroundColor: c.background }]}>
          <Text style={styles.sessionIconText}>💬</Text>
        </View>
        <View style={styles.sessionInfo}>
          <Text style={[styles.sessionSummary, { color: c.textPrimary }]} numberOfLines={1}>
            {item.summary || t('common.emptySession')}
          </Text>
          <View style={styles.sessionMeta}>
            <Text style={[styles.sessionMetaText, { color: c.textTertiary }]}>
              {t('session.messages', { count: item.messageCount })}
            </Text>
            <Text style={[styles.sessionMetaText, { color: c.textTertiary }]}>·</Text>
            <SessionTimeAgo timestamp={item.modifiedAt} />
          </View>
        </View>
        <Text style={[styles.sessionArrow, { color: c.textTertiary }]}>›</Text>
      </TouchableOpacity>
    );
  };

  return (
    <View style={[styles.container, { backgroundColor: c.background }]}>
      {sessions.length === 0 && !loading ? (
        <View style={styles.emptyState}>
          <Text style={[styles.emptyTitle, { color: c.textPrimary }]}>{t('session.empty')}</Text>
          <Text style={[styles.emptyHint, { color: c.textTertiary }]}>
            {t('session.emptyHint')}
          </Text>
        </View>
      ) : (
        <FlatList
          data={sessions}
          renderItem={renderItem}
          keyExtractor={(item) => item.id}
          contentContainerStyle={styles.listContent}
          refreshControl={
            <RefreshControl refreshing={loading} onRefresh={handleRefresh} />
          }
        />
      )}
    </View>
  );
}

// Sub-component to use the hook for each item
function SessionTimeAgo({ timestamp }: { timestamp: number }) {
  const { t } = useTranslation();
  const { theme } = useTheme();
  const c = theme.colors;
  const d = new Date(timestamp * 1000);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  let text: string;
  if (diffMins < 1) text = t('session.justNow');
  else if (diffMins < 60) text = t('session.minutesAgo', { count: diffMins });
  else if (diffHours < 24) text = t('session.hoursAgo', { count: diffHours });
  else if (diffDays < 7) text = t('session.daysAgo', { count: diffDays });
  else text = `${d.getMonth() + 1}/${d.getDate()}`;

  return <Text style={[styles.sessionMetaText, { color: c.textTertiary }]}>{text}</Text>;
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  listContent: {
    padding: 16,
  },
  sessionCard: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 16,
    padding: 16,
    marginBottom: 10,
    borderWidth: 1,
  },
  sessionIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 14,
  },
  sessionIconText: {
    fontSize: 18,
  },
  sessionInfo: {
    flex: 1,
  },
  sessionSummary: {
    fontSize: 15,
    fontWeight: '500',
    marginBottom: 4,
  },
  sessionMeta: {
    flexDirection: 'row',
    gap: 6,
  },
  sessionMetaText: {
    fontSize: 12,
  },
  sessionArrow: {
    fontSize: 22,
    marginLeft: 8,
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
  },
});
