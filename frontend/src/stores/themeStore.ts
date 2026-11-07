import { create } from 'zustand';
import { GetConfig, SaveConfig } from '../api';

export type ThemeMode = 'light' | 'dark' | 'auto';
export type ThemeColor = 'blue' | 'green' | 'purple' | 'orange' | 'pink';

export const COLOR_MAP: Record<ThemeColor, { hex: string; label: string }> = {
  blue: { hex: '#1677ff', label: '极客蓝' },
  green: { hex: '#52c41a', label: '极光绿' },
  purple: { hex: '#722ed1', label: '科技紫' },
  orange: { hex: '#fa8c16', label: '晚霞橙' },
  pink: { hex: '#eb2f96', label: '赛博朋克' },
};

interface ThemeState {
  mode: ThemeMode;
  color: ThemeColor;
  isDark: boolean;
  setThemeMode: (mode: ThemeMode) => Promise<void>;
  setThemeColor: (color: ThemeColor) => Promise<void>;
  syncFromConfig: (mode: string, color: string) => void;
}

function computeIsDark(mode: ThemeMode): boolean {
  if (mode === 'dark') return true;
  if (mode === 'light') return false;
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  return true;
}

function applyCSSVariables(isDark: boolean, color: ThemeColor) {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;
  const primaryHex = COLOR_MAP[color]?.hex || COLOR_MAP.blue.hex;
  root.style.setProperty('--primary-color', primaryHex);
  root.style.setProperty('--bg-color', isDark ? '#0f172a' : '#f5f7fa');
  root.style.setProperty('--card-bg', isDark ? '#1e293b' : '#ffffff');
  root.style.setProperty('--text-color', isDark ? '#f1f5f9' : '#1e293b');
  root.style.setProperty('--muted-color', isDark ? '#94a3b8' : '#64748b');
  root.style.setProperty('--border-color', isDark ? '#334155' : '#e2e8f0');
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: 'dark',
  color: 'blue',
  isDark: true,

  syncFromConfig: (mode, color) => {
    const validMode: ThemeMode =
      mode === 'light' || mode === 'dark' || mode === 'auto' ? mode : 'dark';
    const validColor: ThemeColor =
      color in COLOR_MAP ? (color as ThemeColor) : 'blue';
    const isDark = computeIsDark(validMode);
    applyCSSVariables(isDark, validColor);
    set({ mode: validMode, color: validColor, isDark });
  },

  setThemeMode: async (mode) => {
    const isDark = computeIsDark(mode);
    applyCSSVariables(isDark, get().color);
    set({ mode, isDark });
    const cfg = await GetConfig();
    await SaveConfig({ ...cfg, themeMode: mode });
  },

  setThemeColor: async (color) => {
    applyCSSVariables(get().isDark, color);
    set({ color });
    const cfg = await GetConfig();
    await SaveConfig({ ...cfg, themeColor: color });
  },
}));
