import React from 'react';
import { Segmented, Space, Tooltip } from 'antd';
import {
  useThemeStore,
  COLOR_MAP,
  ThemeColor,
  ThemeMode,
} from '../stores/themeStore';

export const ThemeSwitcher: React.FC = () => {
  const mode = useThemeStore((s) => s.mode);
  const color = useThemeStore((s) => s.color);
  const setThemeMode = useThemeStore((s) => s.setThemeMode);
  const setThemeColor = useThemeStore((s) => s.setThemeColor);

  return (
    <Space size={12} align="center">
      <Segmented
        size="small"
        value={mode}
        onChange={(val) => setThemeMode(val as ThemeMode)}
        options={[
          { label: '浅色', value: 'light' },
          { label: '暗黑', value: 'dark' },
          { label: '跟随系统', value: 'auto' },
        ]}
      />
      <Space size={6}>
        {(Object.keys(COLOR_MAP) as ThemeColor[]).map((key) => {
          const item = COLOR_MAP[key];
          const active = color === key;
          return (
            <Tooltip key={key} title={item.label}>
              <button
                type="button"
                aria-label={item.label}
                onClick={() => setThemeColor(key)}
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: item.hex,
                  border: active ? '2px solid #ffffff' : '2px solid transparent',
                  boxShadow: active ? `0 0 0 2px ${item.hex}` : 'none',
                  cursor: 'pointer',
                  padding: 0,
                  transition: 'transform 0.15s ease',
                  transform: active ? 'scale(1.15)' : 'scale(1)',
                }}
              />
            </Tooltip>
          );
        })}
      </Space>
    </Space>
  );
};
