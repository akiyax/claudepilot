import React, { useState } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, ActivityIndicator } from 'react-native';
import { useTheme } from '../hooks/useTheme';
import { useTranslation } from 'react-i18next';
import { Brand } from '../theme/colors';
import type { ToolCall } from '../types/models';

interface ToolCallCardProps {
  tool: ToolCall;
}

// Tool icon mapping
const TOOL_ICONS: Record<string, string> = {
  Read: '📖',
  Edit: '✏️',
  Write: '📝',
  Bash: '⚡',
  Grep: '🔍',
  Glob: '📂',
  Agent: '🤖',
  WebFetch: '🌐',
  WebSearch: '🔎',
  TodoWrite: '📋',
  NotebookEdit: '📓',
};

function getToolDisplayName(name: string): string {
  // Strip mcp__ prefix for display
  if (name.startsWith('mcp__')) {
    return name.replace('mcp__', '').replace(/_/g, ' ');
  }
  return name;
}

function getInputSummary(name: string, input: any): string {
  if (!input) return '';
  if (name === 'Read' || name === 'Write' || name === 'Edit') {
    const path = input.file_path || input.path || '';
    return path.split('/').pop() || path;
  }
  if (name === 'Bash') {
    const cmd = input.command || '';
    return cmd.length > 60 ? cmd.slice(0, 57) + '...' : cmd;
  }
  if (name === 'Grep') {
    return input.pattern || '';
  }
  if (name === 'Glob') {
    return input.pattern || '';
  }
  return '';
}

export default function ToolCallCard({ tool }: ToolCallCardProps) {
  const [expanded, setExpanded] = useState(false);
  const { theme } = useTheme();
  const { t } = useTranslation();
  const c = theme.colors;
  const icon = TOOL_ICONS[tool.name] || '🔧';

  // Map tool name to theme accent color
  const toolColorMap: Record<string, string> = {
    Read: c.toolRead,
    Edit: c.toolEdit,
    Write: c.toolRead,
    Bash: c.toolBash,
    Grep: c.toolEdit,
    Glob: c.toolRead,
    Agent: c.toolAgent,
  };
  const color = toolColorMap[tool.name] || (tool.name.startsWith('mcp__') ? c.toolMcp : Brand.primary);
  const summary = getInputSummary(tool.name, tool.input);

  return (
    <TouchableOpacity
      style={[styles.container, { backgroundColor: c.toolCardBackground, borderColor: c.surfaceBorder }]}
      onPress={() => setExpanded(!expanded)}
      activeOpacity={0.8}
    >
      <View style={styles.header}>
        <View style={[styles.iconBadge, { backgroundColor: color + '20' }]}>
          <Text style={styles.iconText}>{icon}</Text>
        </View>
        <View style={styles.toolInfo}>
          <View style={styles.nameRow}>
            <Text style={[styles.toolName, { color }]}>
              {getToolDisplayName(tool.name)}
            </Text>
            {tool.status === 'running' && (
              <ActivityIndicator size="small" color={color} />
            )}
            {tool.status === 'completed' && (
              <Text style={[styles.checkMark, { color: c.success }]}>✓</Text>
            )}
            {tool.status === 'failed' && (
              <Text style={[styles.failMark, { color: c.error }]}>✗</Text>
            )}
          </View>
          {summary ? (
            <Text style={[styles.toolSummary, { color: c.textTertiary }]} numberOfLines={1}>{summary}</Text>
          ) : null}
        </View>
      </View>

      {expanded && tool.output && (
        <View style={[styles.outputSection, { borderTopColor: c.surfaceBorder }]}>
          <Text style={[styles.outputLabel, { color: c.textTertiary }]}>{t('common.output')}</Text>
          <Text
            style={[styles.outputText, { color: c.textSecondary }, tool.isError && { color: c.error }]}
            numberOfLines={10}
          >
            {tool.output.length > 500
              ? tool.output.slice(0, 500) + '...'
              : tool.output}
          </Text>
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
    alignItems: 'center',
  },
  iconBadge: {
    width: 32,
    height: 32,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 10,
  },
  iconText: {
    fontSize: 16,
  },
  toolInfo: {
    flex: 1,
  },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  toolName: {
    fontSize: 14,
    fontWeight: '600',
    fontFamily: 'monospace',
  },
  checkMark: {
    fontSize: 14,
    fontWeight: '700',
  },
  failMark: {
    fontSize: 14,
    fontWeight: '700',
  },
  toolSummary: {
    fontSize: 12,
    marginTop: 2,
    fontFamily: 'monospace',
  },
  outputSection: {
    marginTop: 10,
    paddingTop: 10,
    borderTopWidth: 1,
  },
  outputLabel: {
    fontSize: 11,
    fontWeight: '600',
    marginBottom: 4,
  },
  outputText: {
    fontSize: 12,
    fontFamily: 'monospace',
    lineHeight: 18,
  },
});
