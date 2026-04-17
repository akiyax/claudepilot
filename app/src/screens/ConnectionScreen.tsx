import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { useTranslation } from 'react-i18next';
import { useConnectionStore } from '../stores/connectionStore';
import { useTheme } from '../hooks/useTheme';
import { Brand } from '../theme/colors';

export default function ConnectionScreen() {
  const { connectByQR, connectByPairingCode, isConnecting, error } = useConnectionStore();
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

  const [activeTab, setActiveTab] = useState<'pair' | 'manual'>('pair');
  const [pairCode, setPairCode] = useState('');
  const [hostIP, setHostIP] = useState('');
  const [port, setPort] = useState('8077');

  const handlePairConnect = async () => {
    if (!pairCode || pairCode.length !== 6) {
      Alert.alert(t('common.alert'), t('common.pairCodeRequired'));
      return;
    }
    if (!hostIP) {
      Alert.alert(t('common.alert'), t('common.hostIPRequired'));
      return;
    }
    try {
      await connectByPairingCode(hostIP, parseInt(port) || 8077, pairCode);
    } catch (err: any) {
      Alert.alert(t('common.connectFailed'), err.message);
    }
  };

  return (
    <KeyboardAvoidingView
      style={[styles.container, { backgroundColor: c.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <View style={styles.header}>
        <Text style={[styles.title, { color: Brand.primary }]}>ClaudePilot</Text>
        <Text style={[styles.subtitle, { color: c.textSecondary }]}>{t('subtitle')}</Text>
      </View>

      <View style={[styles.tabBar, { backgroundColor: c.surface }]}>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'pair' && { backgroundColor: Brand.primary }]}
          onPress={() => setActiveTab('pair')}
        >
          <Text style={[styles.tabText, { color: c.textSecondary }, activeTab === 'pair' && styles.activeTabText]}>
            {t('connection.pairCode')}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'manual' && { backgroundColor: Brand.primary }]}
          onPress={() => setActiveTab('manual')}
        >
          <Text style={[styles.tabText, { color: c.textSecondary }, activeTab === 'manual' && styles.activeTabText]}>
            {t('connection.manual')}
          </Text>
        </TouchableOpacity>
      </View>

      {activeTab === 'pair' && (
        <View style={styles.form}>
          <Text style={[styles.label, { color: c.textPrimary }]}>{t('connection.hostIP')}</Text>
          <TextInput
            style={[styles.input, { backgroundColor: c.surface, color: c.textPrimary, borderColor: c.surfaceBorder }]}
            placeholder="192.168.x.x"
            placeholderTextColor={c.textTertiary}
            value={hostIP}
            onChangeText={setHostIP}
            keyboardType="numeric"
            autoCapitalize="none"
          />

          <Text style={[styles.label, { color: c.textPrimary }]}>{t('connection.port')}</Text>
          <TextInput
            style={[styles.input, { backgroundColor: c.surface, color: c.textPrimary, borderColor: c.surfaceBorder }]}
            placeholder="8077"
            placeholderTextColor={c.textTertiary}
            value={port}
            onChangeText={setPort}
            keyboardType="numeric"
          />

          <Text style={[styles.label, { color: c.textPrimary }]}>{t('connection.pairCode')}</Text>
          <TextInput
            style={[styles.input, { backgroundColor: c.surface, color: c.textPrimary, borderColor: c.surfaceBorder }]}
            placeholder={t('connection.pairCodeHint')}
            placeholderTextColor={c.textTertiary}
            value={pairCode}
            onChangeText={setPairCode}
            keyboardType="number-pad"
            maxLength={6}
          />

          <Text style={[styles.hint, { color: c.textTertiary }]}>
            {t('connection.hint')}
          </Text>
        </View>
      )}

      {activeTab === 'manual' && (
        <View style={styles.form}>
          <Text style={[styles.hint, { color: c.textTertiary }]}>
            {t('common.manualInDev')}
          </Text>
        </View>
      )}

      {error && (
        <Text style={[styles.errorText, { color: c.error }]}>{error}</Text>
      )}

      <TouchableOpacity
        style={[styles.connectButton, { backgroundColor: Brand.primary }, isConnecting && styles.disabledButton]}
        onPress={handlePairConnect}
        disabled={isConnecting}
      >
        {isConnecting ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.connectButtonText}>{t('connection.connect')}</Text>
        )}
      </TouchableOpacity>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 24,
  },
  header: {
    alignItems: 'center',
    marginTop: 60,
    marginBottom: 40,
  },
  title: {
    fontSize: 32,
    fontWeight: '700',
  },
  subtitle: {
    fontSize: 16,
    marginTop: 8,
  },
  tabBar: {
    flexDirection: 'row',
    borderRadius: 12,
    padding: 4,
    marginBottom: 24,
  },
  tab: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
    borderRadius: 10,
  },
  tabText: {
    fontSize: 14,
    fontWeight: '600',
  },
  activeTabText: {
    color: '#fff',
  },
  form: {
    gap: 12,
  },
  label: {
    fontSize: 14,
    fontWeight: '500',
    marginBottom: 4,
  },
  input: {
    borderRadius: 12,
    padding: 16,
    fontSize: 16,
    borderWidth: 1,
  },
  hint: {
    fontSize: 13,
    textAlign: 'center',
    marginTop: 8,
  },
  errorText: {
    textAlign: 'center',
    marginTop: 12,
    fontSize: 14,
  },
  connectButton: {
    borderRadius: 16,
    paddingVertical: 18,
    alignItems: 'center',
    marginTop: 24,
  },
  disabledButton: {
    opacity: 0.6,
  },
  connectButtonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '600',
  },
});
