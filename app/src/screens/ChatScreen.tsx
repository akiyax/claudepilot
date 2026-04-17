import React, { useState, useRef, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  FlatList,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
} from 'react-native';
import { useChatStore } from '../stores/chatStore';
import { useConnectionStore } from '../stores/connectionStore';
import { colors } from '../theme/colors';
import type { ChatMessage } from '../types/models';

export default function ChatScreen() {
  const {
    messages,
    isStreaming,
    permissionRequest,
    sessionTitle,
    contextUsage,
    sendMessage,
    respondPermission,
    startSession,
  } = useChatStore();

  const { cliVersion } = useConnectionStore();
  const [inputText, setInputText] = useState('');
  const flatListRef = useRef<FlatList>(null);
  const [showNewMessage, setShowNewMessage] = useState(false);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (messages.length > 0) {
      flatListRef.current?.scrollToEnd({ animated: true });
    }
  }, [messages]);

  const handleSend = () => {
    if (!inputText.trim()) return;
    sendMessage(inputText.trim());
    setInputText('');
  };

  const renderMessage = ({ item }: { item: ChatMessage }) => {
    const isUser = item.type === 'user';
    return (
      <View style={[styles.messageRow, isUser ? styles.userRow : styles.assistantRow]}>
        <View style={[styles.messageBubble, isUser ? styles.userBubble : styles.assistantBubble]}>
          <Text style={[styles.messageText, isUser ? styles.userText : styles.assistantText]}>
            {item.content}
          </Text>
          {item.isStreaming && (
            <Text style={styles.streamingCursor}>▊</Text>
          )}
        </View>
      </View>
    );
  };

  // Permission request overlay
  const renderPermissionSheet = () => {
    if (!permissionRequest) return null;
    return (
      <View style={styles.permissionSheet}>
        <Text style={styles.permissionTitle}>权限请求</Text>
        <Text style={styles.permissionTool}>
          {permissionRequest.toolName}
        </Text>
        {permissionRequest.displayText && (
          <Text style={styles.permissionDesc}>
            {permissionRequest.displayText}
          </Text>
        )}
        <View style={styles.permissionButtons}>
          <TouchableOpacity
            style={[styles.permButton, styles.denyButton]}
            onPress={() => respondPermission(permissionRequest.requestId, 'deny')}
          >
            <Text style={styles.denyButtonText}>拒绝</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.permButton, styles.allowButton]}
            onPress={() => respondPermission(permissionRequest.requestId, 'allow')}
          >
            <Text style={styles.allowButtonText}>允许</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={90}
    >
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>{sessionTitle}</Text>
        {contextUsage.contextWindow > 0 && (
          <Text style={styles.contextUsage}>
            {Math.round(contextUsage.totalTokens / 1000)}k/{Math.round(contextUsage.contextWindow / 1000)}k
          </Text>
        )}
      </View>

      {/* Messages */}
      <FlatList
        ref={flatListRef}
        data={messages}
        renderItem={renderMessage}
        keyExtractor={(item) => item.id}
        style={styles.messageList}
        contentContainerStyle={styles.messageListContent}
        onScroll={(e) => {
          // Show "new message" indicator when not at bottom
        }}
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyTitle}>开始新对话</Text>
            <Text style={styles.emptyHint}>
              发送消息开始与 Claude 对话
            </Text>
            <TouchableOpacity
              style={styles.startButton}
              onPress={() => startSession({})}
            >
              <Text style={styles.startButtonText}>新建会话</Text>
            </TouchableOpacity>
          </View>
        }
      />

      {/* Permission Sheet */}
      {renderPermissionSheet()}

      {/* Input Bar */}
      <View style={styles.inputBar}>
        <TextInput
          style={styles.textInput}
          placeholder="输入消息..."
          placeholderTextColor={colors.light.textTertiary}
          value={inputText}
          onChangeText={setInputText}
          multiline
          maxLength={10000}
          editable={!isStreaming}
        />
        <TouchableOpacity
          style={[styles.sendButton, (!inputText.trim() || isStreaming) && styles.sendDisabled]}
          onPress={handleSend}
          disabled={!inputText.trim() || isStreaming}
        >
          {isStreaming ? (
            <ActivityIndicator size="small" color="#fff" />
          ) : (
            <Text style={styles.sendButtonText}>▶</Text>
          )}
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.light.background,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: colors.light.border,
    backgroundColor: colors.light.card,
  },
  headerTitle: {
    fontSize: 17,
    fontWeight: '600',
    color: colors.light.textPrimary,
    flex: 1,
  },
  contextUsage: {
    fontSize: 12,
    color: colors.light.textTertiary,
    fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
  },
  messageList: {
    flex: 1,
  },
  messageListContent: {
    paddingVertical: 12,
  },
  messageRow: {
    paddingHorizontal: 16,
    marginVertical: 4,
  },
  userRow: {
    alignItems: 'flex-end',
  },
  assistantRow: {
    alignItems: 'flex-start',
  },
  messageBubble: {
    maxWidth: '80%',
    borderRadius: 18,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  userBubble: {
    backgroundColor: colors.light.userBubble,
  },
  assistantBubble: {
    backgroundColor: colors.light.assistantBubble,
    borderWidth: 1,
    borderColor: colors.light.border,
  },
  messageText: {
    fontSize: 15,
    lineHeight: 22,
  },
  userText: {
    color: colors.light.textPrimary,
  },
  assistantText: {
    color: colors.light.textPrimary,
  },
  streamingCursor: {
    color: colors.light.primary,
    fontSize: 14,
  },
  emptyState: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingTop: 120,
  },
  emptyTitle: {
    fontSize: 22,
    fontWeight: '700',
    color: colors.light.textPrimary,
    marginBottom: 8,
  },
  emptyHint: {
    fontSize: 15,
    color: colors.light.textTertiary,
    marginBottom: 24,
  },
  startButton: {
    backgroundColor: colors.light.primary,
    borderRadius: 16,
    paddingHorizontal: 32,
    paddingVertical: 14,
  },
  startButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  permissionSheet: {
    backgroundColor: colors.light.card,
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 20,
    borderWidth: 1,
    borderColor: colors.light.border,
  },
  permissionTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: colors.light.textPrimary,
    marginBottom: 8,
  },
  permissionTool: {
    fontSize: 14,
    fontWeight: '600',
    color: colors.light.primary,
    marginBottom: 4,
    fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
  },
  permissionDesc: {
    fontSize: 14,
    color: colors.light.textSecondary,
    marginBottom: 16,
  },
  permissionButtons: {
    flexDirection: 'row',
    gap: 12,
  },
  permButton: {
    flex: 1,
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
  },
  denyButton: {
    backgroundColor: colors.light.error + '20',
    borderWidth: 1,
    borderColor: colors.light.error,
  },
  allowButton: {
    backgroundColor: colors.light.primary,
  },
  denyButtonText: {
    color: colors.light.error,
    fontWeight: '600',
    fontSize: 16,
  },
  allowButtonText: {
    color: '#fff',
    fontWeight: '600',
    fontSize: 16,
  },
  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 12,
    paddingVertical: 8,
    backgroundColor: colors.light.card,
    borderTopWidth: 1,
    borderTopColor: colors.light.border,
    gap: 8,
  },
  textInput: {
    flex: 1,
    minHeight: 40,
    maxHeight: 120,
    backgroundColor: colors.light.background,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 16,
    color: colors.light.textPrimary,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: colors.light.primary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendDisabled: {
    opacity: 0.5,
  },
  sendButtonText: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '700',
  },
});
