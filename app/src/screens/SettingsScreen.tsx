import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useConnectionStore } from '../stores/connectionStore';
import { colors } from '../theme/colors';

export default function SettingsScreen() {
  const { connection, daemonVersion, cliVersion, disconnect } = useConnectionStore();

  return (
    <View style={styles.container}>
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>连接信息</Text>
        <View style={styles.row}>
          <Text style={styles.label}>状态</Text>
          <Text style={[styles.value, { color: connection.connected ? colors.light.success : colors.light.error }]}>
            {connection.connected ? '已连接' : '未连接'}
          </Text>
        </View>
        {daemonVersion && (
          <View style={styles.row}>
            <Text style={styles.label}>Daemon 版本</Text>
            <Text style={styles.value}>{daemonVersion}</Text>
          </View>
        )}
        {cliVersion && (
          <View style={styles.row}>
            <Text style={styles.label}>CLI 版本</Text>
            <Text style={styles.value}>{cliVersion}</Text>
          </View>
        )}
        {connection.daemonId && (
          <View style={styles.row}>
            <Text style={styles.label}>Daemon ID</Text>
            <Text style={styles.value} numberOfLines={1}>{connection.daemonId}</Text>
          </View>
        )}
      </View>

      <TouchableOpacity style={styles.disconnectButton} onPress={disconnect}>
        <Text style={styles.disconnectButtonText}>断开连接</Text>
      </TouchableOpacity>

      <Text style={styles.version}>ClaudePilot v0.1.0</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.light.background, padding: 16 },
  section: { backgroundColor: colors.light.card, borderRadius: 16, padding: 16, marginBottom: 16 },
  sectionTitle: { fontSize: 14, fontWeight: '600', color: colors.light.textTertiary, marginBottom: 12, textTransform: 'uppercase' },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 8, borderBottomWidth: 1, borderBottomColor: colors.light.border },
  label: { fontSize: 15, color: colors.light.textSecondary },
  value: { fontSize: 15, color: colors.light.textPrimary, fontWeight: '500' },
  disconnectButton: { backgroundColor: colors.light.error + '15', borderRadius: 16, paddingVertical: 16, alignItems: 'center', marginTop: 8 },
  disconnectButtonText: { color: colors.light.error, fontSize: 16, fontWeight: '600' },
  version: { fontSize: 12, color: colors.light.textTertiary, textAlign: 'center', marginTop: 24 },
});
