import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useTheme } from '../hooks/useTheme';
import { useTranslation } from 'react-i18next';

interface ThinkingPanelProps {
  content: string;
  isStreaming?: boolean;
}

export default function ThinkingPanel({ content, isStreaming }: ThinkingPanelProps) {
  const [expanded, setExpanded] = useState(false);
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

  if (!content) return null;

  const tokenEstimate = Math.ceil(content.length / 4);

  return (
    <TouchableOpacity
      style={[styles.container, { backgroundColor: c.thinkingBackground, borderColor: c.thinkingBorder }]}
      onPress={() => setExpanded(!expanded)}
      activeOpacity={0.8}
    >
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Text style={styles.icon}>💭</Text>
          <Text style={[styles.title, { color: c.toolBash }]}>
            {isStreaming ? t('chat.thinking') : t('chat.thought')}
          </Text>
        </View>
        <Text style={[styles.tokenCount, { color: c.textTertiary }]}>
          ~{tokenEstimate} tokens
        </Text>
      </View>
      {expanded && (
        <View style={[styles.body, { borderTopColor: c.thinkingBorder }]}>
          <Text style={[styles.bodyText, { color: c.textSecondary }]}>{content}</Text>
        </View>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    borderRadius: 12,
    padding: 12,
    marginHorizontal: 16,
    marginVertical: 4,
    borderWidth: 1,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  icon: {
    fontSize: 14,
  },
  title: {
    fontSize: 13,
    fontWeight: '600',
  },
  tokenCount: {
    fontSize: 11,
  },
  body: {
    marginTop: 10,
    paddingTop: 10,
    borderTopWidth: 1,
  },
  bodyText: {
    fontSize: 13,
    lineHeight: 20,
  },
});
