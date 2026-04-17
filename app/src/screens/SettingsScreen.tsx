import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useTranslation } from 'react-i18next';
import { useConnectionStore } from '../stores/connectionStore';
import { useTheme } from '../hooks/useTheme';

export default function SettingsScreen() {
  const { connection, daemonVersion, cliVersion, disconnect } = useConnectionStore();
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

  return (
    <View style={[styles.container, { backgroundColor: c.background }]}>
      <View style={[styles.section, { backgroundColor: c.surface }]}>
        <Text style={[styles.sectionTitle, { color: c.textTertiary }]}>
          {t('settings.connectionInfo')}
        </Text>
        <View style={[styles.row, { borderBottomColor: c.surfaceBorder }]}>
          <Text style={[styles.label, { color: c.textSecondary }]}>{t('settings.status')}</Text>
          <Text style={[styles.value, { color: connection.connected ? c.success : c.error, fontWeight: '500' }]}>
            {connection.connected ? t('settings.connected') : t('settings.disconnected')}
          </Text>
        </View>
        {daemonVersion && (
          <View style={[styles.row, { borderBottomColor: c.surfaceBorder }]}>
            <Text style={[styles.label, { color: c.textSecondary }]}>{t('settings.daemonVersion')}</Text>
            <Text style={[styles.value, { color: c.textPrimary }]}>{daemonVersion}</Text>
          </View>
        )}
        {cliVersion && (
          <View style={[styles.row, { borderBottomColor: c.surfaceBorder }]}>
            <Text style={[styles.label, { color: c.textSecondary }]}>{t('settings.cliVersion')}</Text>
            <Text style={[styles.value, { color: c.textPrimary }]}>{cliVersion}</Text>
          </View>
        )}
        {connection.daemonId && (
          <View style={[styles.row, { borderBottomColor: c.surfaceBorder }]}>
            <Text style={[styles.label, { color: c.textSecondary }]}>{t('settings.daemonId')}</Text>
            <Text style={[styles.value, { color: c.textPrimary }]} numberOfLines={1}>{connection.daemonId}</Text>
          </View>
        )}
      </View>

      <TouchableOpacity
        style={[styles.disconnectButton, { backgroundColor: c.error + '15' }]}
        onPress={disconnect}
      >
        <Text style={[styles.disconnectButtonText, { color: c.error }]}>
          {t('settings.disconnect')}
        </Text>
      </TouchableOpacity>

      <Text style={[styles.version, { color: c.textTertiary }]}>ClaudePilot v0.1.0</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 16 },
  section: { borderRadius: 16, padding: 16, marginBottom: 16 },
  sectionTitle: { fontSize: 14, fontWeight: '600', marginBottom: 12, textTransform: 'uppercase' },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 8, borderBottomWidth: 1 },
  label: { fontSize: 15 },
  value: { fontSize: 15 },
  disconnectButton: { borderRadius: 16, paddingVertical: 16, alignItems: 'center', marginTop: 8 },
  disconnectButtonText: { fontSize: 16, fontWeight: '600' },
  version: { fontSize: 12, textAlign: 'center', marginTop: 24 },
});
