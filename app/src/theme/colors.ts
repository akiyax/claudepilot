/**
 * ClaudePilot Theme Colors
 *
 * Design language aligned with Claude official app:
 * Warm, minimalist, restrained. Generous whitespace, soft colors, clear hierarchy.
 *
 * Light mode uses Claude's signature Pampas (#F4F3EE) warm-white background.
 * Dark mode uses near-black with blue-tinted user bubbles.
 */

// ─── Brand Colors ────────────────────────────────────────

export const Brand = {
  /** Crail — Terracotta orange, primary brand color */
  primary: '#C15F3C',
  /** Crail lightened for dark mode */
  primaryLight: '#D4754E',
  /** Cloudy — Neutral gray for secondary elements */
  neutral: '#B1ADA1',
  /** Cloudy lightened for dark mode */
  neutralLight: '#8E8E93',
} as const;

// ─── Light Mode ──────────────────────────────────────────

export const LightColors = {
  // Backgrounds
  background: '#F4F3EE',       // Pampas — warm white page bg
  surface: '#FFFFFF',           // Cards, Claude replies
  surfaceBorder: '#E8E8E8',    // Card borders
  inputBackground: '#E6F0FF',  // Input bar bg (light blue tint)
  inputBorder: '#D0D5DD',      // Input border

  // Text
  textPrimary: '#333333',      // Deep charcoal
  textSecondary: '#666666',    // Medium gray
  textTertiary: '#999999',     // Light gray (timestamps, hints)
  textInverse: '#FFFFFF',      // White on brand color

  // Chat bubbles
  userBubble: '#F0F0F0',       // Light gray
  userBubbleText: '#333333',
  assistantBubble: '#FFFFFF',  // White
  assistantBubbleBorder: '#E8E8E8',

  // Thinking panel
  thinkingBackground: '#F0EEFF',
  thinkingBorder: '#D4CCFF',

  // Tool cards
  toolCardBackground: '#F8F8F6',

  // Dividers
  divider: '#E5E5E5',

  // Status
  success: '#34C759',
  warning: '#FF9500',
  error: '#FF3B30',

  // Tool type accent colors
  toolRead: '#4A90D9',
  toolEdit: '#E8A838',
  toolBash: '#7B68EE',
  toolAgent: '#34C759',
  toolMcp: '#FF6B6B',
} as const;

// ─── Dark Mode ───────────────────────────────────────────

export const DarkColors = {
  // Backgrounds
  background: '#1C1C1E',       // Near-black with warm undertone
  surface: '#2C2C2E',          // Dark gray cards
  surfaceBorder: '#3A3A3C',    // Dark border
  inputBackground: '#1C1C2E',  // Deep indigo tint
  inputBorder: '#3A3A3C',

  // Text
  textPrimary: '#E5E5EA',      // Light gray-white
  textSecondary: '#A1A1A6',    // Medium light gray
  textTertiary: '#636366',     // Dark gray
  textInverse: '#1C1C1E',

  // Chat bubbles
  userBubble: '#2E3A52',       // Deep indigo blue
  userBubbleText: '#E5E5EA',
  assistantBubble: '#2C2C2E',  // Dark gray
  assistantBubbleBorder: '#3A3A3C',

  // Thinking panel
  thinkingBackground: '#251F3A',
  thinkingBorder: '#3D2E6B',

  // Tool cards
  toolCardBackground: '#242426',

  // Dividers
  divider: '#38383A',

  // Status
  success: '#30D158',
  warning: '#FF9F0A',
  error: '#FF453A',

  // Tool type accent colors (brightened for dark bg)
  toolRead: '#5BA3E8',
  toolEdit: '#F0B84A',
  toolBash: '#9485F0',
  toolAgent: '#30D158',
  toolMcp: '#FF453A',
} as const;

// ─── Theme Type ──────────────────────────────────────────

export type ThemeColors = typeof LightColors | typeof DarkColors;

export interface Theme {
  colors: typeof LightColors | typeof DarkColors;
  isDark: boolean;
}

// ─── Theme Builder ───────────────────────────────────────

export function createTheme(isDark: boolean): Theme {
  return {
    colors: isDark ? DarkColors : LightColors,
    isDark,
  };
}

// ─── Convenience flat colors for StyleSheet usage ─────────
// These provide quick access for light mode (default).
// For dark mode, use createTheme() with a React context.

export const colors = {
  light: {
    primary: Brand.primary,
    background: LightColors.background,
    card: LightColors.surface,
    border: LightColors.surfaceBorder,
    textPrimary: LightColors.textPrimary,
    textSecondary: LightColors.textSecondary,
    textTertiary: LightColors.textTertiary,
    userBubble: LightColors.userBubble,
    assistantBubble: LightColors.surface,
    success: '#34C759',
    error: '#FF3B30',
  },
  dark: {
    primary: Brand.primaryLight,
    background: DarkColors.background,
    card: DarkColors.surface,
    border: DarkColors.surfaceBorder,
    textPrimary: DarkColors.textPrimary,
    textSecondary: DarkColors.textSecondary,
    textTertiary: DarkColors.textTertiary,
    userBubble: DarkColors.userBubble,
    assistantBubble: DarkColors.surface,
    success: '#30D158',
    error: '#FF453A',
  },
} as const;
