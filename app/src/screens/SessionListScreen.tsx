import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { colors } from '../theme/colors';

export default function SessionListScreen() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>会话列表</Text>
      <Text style={styles.hint}>会话管理功能开发中...</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.light.background, justifyContent: 'center', alignItems: 'center' },
  title: { fontSize: 20, fontWeight: '700', color: colors.light.textPrimary, marginBottom: 8 },
  hint: { fontSize: 14, color: colors.light.textTertiary },
});
