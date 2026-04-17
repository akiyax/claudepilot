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
import { useConnectionStore } from '../stores/connectionStore';
import { colors } from '../theme/colors';

export default function ConnectionScreen() {
  const { connectByQR, connectByPairingCode, isConnecting, error } = useConnectionStore();
  const [activeTab, setActiveTab] = useState<'pair' | 'manual'>('pair');
  const [pairCode, setPairCode] = useState('');
  const [hostIP, setHostIP] = useState('');
  const [port, setPort] = useState('8077');

  const handlePairConnect = async () => {
    if (!pairCode || pairCode.length !== 6) {
      Alert.alert('提示', '请输入 6 位配对码');
      return;
    }
    if (!hostIP) {
      Alert.alert('提示', '请输入电脑 IP 地址');
      return;
    }
    try {
      await connectByPairingCode(hostIP, parseInt(port) || 8077, pairCode);
    } catch (err: any) {
      Alert.alert('连接失败', err.message);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <View style={styles.header}>
        <Text style={styles.title}>ClaudePilot</Text>
        <Text style={styles.subtitle}>远程控制 Claude Code</Text>
      </View>

      <View style={styles.tabBar}>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'pair' && styles.activeTab]}
          onPress={() => setActiveTab('pair')}
        >
          <Text style={[styles.tabText, activeTab === 'pair' && styles.activeTabText]}>
            配对码
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'manual' && styles.activeTab]}
          onPress={() => setActiveTab('manual')}
        >
          <Text style={[styles.tabText, activeTab === 'manual' && styles.activeTabText]}>
            手动输入
          </Text>
        </TouchableOpacity>
      </View>

      {activeTab === 'pair' && (
        <View style={styles.form}>
          <Text style={styles.label}>电脑 IP 地址</Text>
          <TextInput
            style={styles.input}
            placeholder="192.168.x.x"
            value={hostIP}
            onChangeText={setHostIP}
            keyboardType="numeric"
            autoCapitalize="none"
          />

          <Text style={styles.label}>端口</Text>
          <TextInput
            style={styles.input}
            placeholder="8077"
            value={port}
            onChangeText={setPort}
            keyboardType="numeric"
          />

          <Text style={styles.label}>配对码</Text>
          <TextInput
            style={styles.input}
            placeholder="6 位数字"
            value={pairCode}
            onChangeText={setPairCode}
            keyboardType="number-pad"
            maxLength={6}
          />

          <Text style={styles.hint}>
            在电脑终端运行 claudepilot 查看配对码
          </Text>
        </View>
      )}

      {activeTab === 'manual' && (
        <View style={styles.form}>
          <Text style={styles.hint}>
            手动输入功能开发中...
          </Text>
        </View>
      )}

      {error && (
        <Text style={styles.errorText}>{error}</Text>
      )}

      <TouchableOpacity
        style={[styles.connectButton, isConnecting && styles.disabledButton]}
        onPress={handlePairConnect}
        disabled={isConnecting}
      >
        {isConnecting ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.connectButtonText}>连接</Text>
        )}
      </TouchableOpacity>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.light.background,
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
    color: colors.light.primary,
  },
  subtitle: {
    fontSize: 16,
    color: colors.light.textSecondary,
    marginTop: 8,
  },
  tabBar: {
    flexDirection: 'row',
    backgroundColor: colors.light.card,
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
  activeTab: {
    backgroundColor: colors.light.primary,
  },
  tabText: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.light.textSecondary,
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
    color: colors.light.textPrimary,
    marginBottom: 4,
  },
  input: {
    backgroundColor: colors.light.card,
    borderRadius: 12,
    padding: 16,
    fontSize: 16,
    color: colors.light.textPrimary,
    borderWidth: 1,
    borderColor: colors.light.border,
  },
  hint: {
    fontSize: 13,
    color: colors.light.textTertiary,
    textAlign: 'center',
    marginTop: 8,
  },
  errorText: {
    color: colors.light.error,
    textAlign: 'center',
    marginTop: 12,
    fontSize: 14,
  },
  connectButton: {
    backgroundColor: colors.light.primary,
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
