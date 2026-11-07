import React, { useMemo, useState } from 'react';
import { Button, Card, Empty, Input, Segmented, Space, Tag, message } from 'antd';
import { ClearOutlined, CopyOutlined } from '@ant-design/icons';
import { useTrafficStore } from '../stores/trafficStore';
import { useThemeStore } from '../stores/themeStore';
import { ClipboardSetText } from '../api';

export const Logs: React.FC = () => {
  const logs = useTrafficStore((s) => s.logs);
  const clearAllLogs = useTrafficStore((s) => s.clearAllLogs);
  const isDark = useThemeStore((s) => s.isDark);

  const [levelFilter, setLevelFilter] = useState<string>('ALL');
  const [keyword, setKeyword] = useState<string>('');

  const filteredLogs = useMemo(() => {
    return logs.filter((line) => {
      if (levelFilter !== 'ALL' && !line.includes(levelFilter)) {
        return false;
      }
      if (keyword.trim() && !line.toLowerCase().includes(keyword.toLowerCase())) {
        return false;
      }
      return true;
    });
  }, [logs, levelFilter, keyword]);

  return (
    <Card
      size="small"
      title={
        <Space>
          <span>实时内核运行日志 (已自动脱敏 UUID / 密码 / Token)</span>
          <Tag>{filteredLogs.length} 条</Tag>
        </Space>
      }
      extra={
        <Space>
          <Segmented
            size="small"
            value={levelFilter}
            onChange={(v) => setLevelFilter(String(v))}
            options={['ALL', 'INFO', 'WARN', 'ERROR']}
          />
          <Input.Search
            size="small"
            allowClear
            placeholder="过滤日志关键字..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ width: 180 }}
          />
          <Button
            size="small"
            icon={<CopyOutlined />}
            onClick={async () => {
              await ClipboardSetText(filteredLogs.join('\n'));
              message.success('日志已复制到剪贴板');
            }}
          >
            复制
          </Button>
          <Button
            size="small"
            danger
            icon={<ClearOutlined />}
            onClick={async () => {
              await clearAllLogs();
              message.info('日志已清空');
            }}
          >
            清空
          </Button>
        </Space>
      }
    >
      <div
        style={{
          height: 500,
          overflowY: 'auto',
          padding: '12px 14px',
          borderRadius: 8,
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontSize: 12.5,
          lineHeight: 1.65,
          background: isDark ? '#090d16' : '#f8fafc',
          color: isDark ? '#e2e8f0' : '#1e293b',
          border: `1px solid ${isDark ? '#1e293b' : '#e2e8f0'}`,
        }}
      >
        {filteredLogs.length === 0 ? (
          <Empty description="暂无匹配的内核日志" style={{ marginTop: 120 }} />
        ) : (
          filteredLogs.map((line, idx) => {
            const isErr = line.includes('ERROR');
            const isWarn = line.includes('WARN');
            return (
              <div
                key={idx}
                style={{
                  color: isErr
                    ? '#ff4d4f'
                    : isWarn
                    ? '#faad14'
                    : isDark
                    ? '#cbd5e1'
                    : '#334155',
                  wordBreak: 'break-all',
                }}
              >
                {line}
              </div>
            );
          })
        )}
      </div>
    </Card>
  );
};
