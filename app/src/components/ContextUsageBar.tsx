import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useTheme } from '../hooks/useTheme';
import type { ContextUsage } from '../types/models';

interface ContextUsageBarProps {
  usage: ContextUsage;
}

export default function ContextUsageBar({ usage }: ContextUsageBarProps) {
  const { theme } = useTheme();
  const c = theme.colors;

  if (usage.contextWindow <= 0) return null;

  const percent = usage.usedPercent || 0;
  const usedK = Math.round(usage.totalTokens / 1000);
  const totalK = Math.round(usage.contextWindow / 1000);

  // Dynamic color based on usage percentage
  let fillColor: string;
  if (percent < 50) {
    fillColor = c.success;
  } else if (percent < 80) {
    fillColor = c.warning;
  } else {
    fillColor = c.error;
  }

  return (
    <View style={styles.container}>
      <View style={[styles.bar, { backgroundColor: c.surfaceBorder }]}>
        <View
          style={[
            styles.fill,
            {
              width: `${Math.min(percent, 100)}%`,
              backgroundColor: fillColor,
            },
          ]}
        />
      </View>
      <Text style={[styles.label, { color: c.textTertiary }]}>
        {usedK}k/{totalK}k
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  bar: {
    width: 48,
    height: 4,
    borderRadius: 2,
    overflow: 'hidden',
  },
  fill: {
    height: '100%',
    borderRadius: 2,
  },
  label: {
    fontSize: 11,
    fontFamily: 'monospace',
  },
});
