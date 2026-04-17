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
import { useTranslation } from 'react-i18next';
import { useChatStore } from '../stores/chatStore';
import { useConnectionStore } from '../stores/connectionStore';
import { useTheme } from '../hooks/useTheme';
import ThinkingPanel from '../components/ThinkingPanel';
import ToolCallCard from '../components/ToolCallCard';
import ContextUsageBar from '../components/ContextUsageBar';
import type { ChatMessage } from '../types/models';

export default function ChatScreen() {
  const {
    messages,
    isStreaming,
    currentThinkingText,
    activeToolCalls,
    permissionRequest,
    sessionTitle,
    contextUsage,
    sendMessage,
    respondPermission,
    startSession,
  } = useChatStore();

  const { cliVersion } = useConnectionStore();
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;

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
        <View style={[
          styles.messageBubble,
          isUser
            ? { backgroundColor: c.userBubble }
            : { backgroundColor: c.assistantBubble, borderColor: c.assistantBubbleBorder, borderWidth: 1 },
        ]}>
          <Text style={[styles.messageText, { color: c.textPrimary }]}>
            {item.content}
          </Text>
          {item.isStreaming && (
            <Text style={[styles.streamingCursor, { color: Brand.primary }]}>▊</Text>
          )}
        </View>
      </View>
    );
  };

  // Permission request overlay
  const renderPermissionSheet = () => {
    if (!permissionRequest) return null;
    return (
      <View style={[styles.permissionSheet, { backgroundColor: c.surface, borderColor: c.surfaceBorder }]}>
        <Text style={[styles.permissionTitle, { color: c.textPrimary }]}>
          {t('chat.permissionTitle')}
        </Text>
        <Text style={[styles.permissionTool, { color: Brand.primary }]}>
          {permissionRequest.toolName}
        </Text>
        {permissionRequest.displayText && (
          <Text style={[styles.permissionDesc, { color: c.textSecondary }]}>
            {permissionRequest.displayText}
          </Text>
        )}
        <View style={styles.permissionButtons}>
          <TouchableOpacity
            style={[styles.permButton, { backgroundColor: c.error + '20', borderColor: c.error, borderWidth: 1 }]}
            onPress={() => respondPermission(permissionRequest.requestId, 'deny')}
          >
            <Text style={[styles.denyButtonText, { color: c.error }]}>{t('chat.deny')}</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.permButton, { backgroundColor: Brand.primary }]}
            onPress={() => respondPermission(permissionRequest.requestId, 'allow')}
          >
            <Text style={styles.allowButtonText}>{t('chat.allow')}</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  };

  return (
    <KeyboardAvoidingView
      style={[styles.container, { backgroundColor: c.background }]}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={90}
    >
      {/* Header */}
      <View style={[styles.header, { borderBottomColor: c.surfaceBorder, backgroundColor: c.surface }]}>
        <Text style={[styles.headerTitle, { color: c.textPrimary }]}>{sessionTitle || t('chat.newConversation')}</Text>
        {contextUsage.contextWindow > 0 && (
          <ContextUsageBar usage={contextUsage} />
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
            <Text style={[styles.emptyTitle, { color: c.textPrimary }]}>{t('chat.newConversation')}</Text>
            <Text style={[styles.emptyHint, { color: c.textTertiary }]}>
              {t('chat.startHint')}
            </Text>
            <TouchableOpacity
              style={[styles.startButton, { backgroundColor: Brand.primary }]}
              onPress={() => startSession({})}
            >
              <Text style={styles.startButtonText}>{t('chat.newSession')}</Text>
            </TouchableOpacity>
          </View>
        }
      />

      {/* Thinking Panel */}
      {currentThinkingText && (
        <ThinkingPanel content={currentThinkingText} isStreaming={isStreaming} />
      )}

      {/* Active Tool Calls */}
      {Array.from(activeToolCalls.values()).map((tool) => (
        <ToolCallCard key={tool.id} tool={tool} />
      ))}

      {/* Permission Sheet */}
      {renderPermissionSheet()}

      {/* Input Bar */}
      <View style={[styles.inputBar, { backgroundColor: c.surface, borderTopColor: c.surfaceBorder }]}>
        <TextInput
          style={[styles.textInput, { backgroundColor: c.background, color: c.textPrimary }]}
          placeholder={t('chat.inputPlaceholder')}
          placeholderTextColor={c.textTertiary}
          value={inputText}
          onChangeText={setInputText}
          multiline
          maxLength={10000}
          editable={!isStreaming}
        />
        <TouchableOpacity
          style={[styles.sendButton, { backgroundColor: Brand.primary }, (!inputText.trim() || isStreaming) && styles.sendDisabled]}
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

import { Brand } from '../theme/colors';

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
  },
  headerTitle: {
    fontSize: 17,
    fontWeight: '600',
    flex: 1,
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
  messageText: {
    fontSize: 15,
    lineHeight: 22,
  },
  streamingCursor: {
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
    marginBottom: 8,
  },
  emptyHint: {
    fontSize: 15,
    marginBottom: 24,
  },
  startButton: {
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
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 20,
    borderWidth: 1,
  },
  permissionTitle: {
    fontSize: 18,
    fontWeight: '700',
    marginBottom: 8,
  },
  permissionTool: {
    fontSize: 14,
    fontWeight: '600',
    marginBottom: 4,
    fontFamily: Platform.OS === 'ios' ? 'Menlo' : 'monospace',
  },
  permissionDesc: {
    fontSize: 14,
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
  denyButtonText: {
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
    borderTopWidth: 1,
    gap: 8,
  },
  textInput: {
    flex: 1,
    minHeight: 40,
    maxHeight: 120,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 16,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
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
