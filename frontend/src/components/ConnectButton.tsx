import React from 'react';
import { Badge, message, Tag, Tooltip } from 'antd';
import {
  PoweroffOutlined,
  LoadingOutlined,
  ThunderboltFilled,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useConfigStore } from '../stores/configStore';
import { useNodeStore } from '../stores/nodeStore';
import { useThemeStore, COLOR_MAP } from '../stores/themeStore';

export const ConnectButton: React.FC = () => {
  const status = useConfigStore((s) => s.status);
  const appConfig = useConfigStore((s) => s.appConfig);
  const lastError = useConfigStore((s) => s.lastError);
  const toggleConnection = useConfigStore((s) => s.toggleConnection);
  const nodes = useNodeStore((s) => s.nodes);
  const themeColor = useThemeStore((s) => s.color);
  const isDark = useThemeStore((s) => s.isDark);

  const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';
  const activeNode =
    nodes.find((n) => n.id === appConfig.activeNodeId) || nodes[0];

  const isConnected = status === 'connected';
  const isBusy = status === 'connecting' || status === 'disconnecting';
  const isError = status === 'error';

  const handleToggle = async () => {
    if (isBusy) return;
    if (!activeNode && !isConnected) {
      message.warning('请先在「节点与订阅」页面导入或选择一个代理节点');
      return;
    }
    try {
      await toggleConnection();
      if (status !== 'connected') {
        message.success(`已连接到 ${activeNode?.name || '代理节点'}`);
      } else {
        message.info('已断开代理连接并还原系统网络');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      message.error(`连接操作失败: ${msg}`);
    }
  };

  const statusMeta = {
    disconnected: {
      badge: 'default' as const,
      label: '未连接 · 点击左侧大按钮立即开启代理',
      btnText: '点击连接',
      ringColor: isDark ? '#334155' : '#cbd5e1',
    },
    connecting: {
      badge: 'processing' as const,
      label: '正在预检配置并启动内核...',
      btnText: '连接中...',
      ringColor: primaryHex,
    },
    connected: {
      badge: 'success' as const,
      label: '代理运行中 · 浏览器与全局流量已受保护',
      btnText: '已连接 (点击断开)',
      ringColor: primaryHex,
    },
    disconnecting: {
      badge: 'warning' as const,
      label: '正在安全停止并还原系统代理...',
      btnText: '断开中...',
      ringColor: '#faad14',
    },
    error: {
      badge: 'error' as const,
      label: lastError || '连接异常，请检查内核日志',
      btnText: '重试连接',
      ringColor: '#ff4d4f',
    },
  }[status];

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'nowrap',
        gap: 20,
        padding: '20px 26px',
        borderRadius: 16,
        background: isDark
          ? `linear-gradient(135deg, rgba(30,41,59,0.95) 0%, rgba(15,23,42,0.92) 100%)`
          : `linear-gradient(135deg, #ffffff 0%, #f8fafc 100%)`,
        border: `1px solid ${
          isConnected ? primaryHex + '66' : isDark ? '#334155' : '#e2e8f0'
        }`,
        boxShadow: isConnected
          ? `0 12px 32px -8px ${primaryHex}33`
          : '0 4px 16px rgba(0,0,0,0.06)',
        transition: 'all 0.3s ease',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 22,
          flex: 1,
          minWidth: 0,
        }}
      >
        <Tooltip title={isConnected ? '点击立即断开代理' : '点击立即启动代理'}>
          <button
            type="button"
            onClick={handleToggle}
            disabled={isBusy}
            className={isConnected ? 'connect-orb connected-pulse' : 'connect-orb'}
            style={{
              width: 88,
              height: 88,
              minWidth: 88,
              minHeight: 88,
              maxWidth: 88,
              maxHeight: 88,
              flex: '0 0 88px',
              flexShrink: 0,
              aspectRatio: '1 / 1',
              padding: 0,
              borderRadius: '50%',
              border: `3px solid ${statusMeta.ringColor}`,
              background: isConnected
                ? `radial-gradient(circle at 30% 30%, ${primaryHex}, ${primaryHex}cc)`
                : isError
                ? 'rgba(255, 77, 79, 0.15)'
                : isDark
                ? '#0f172a'
                : '#f1f5f9',
              color: isConnected
                ? '#ffffff'
                : isError
                ? '#ff4d4f'
                : isDark
                ? '#cbd5e1'
                : '#475569',
              cursor: isBusy ? 'wait' : 'pointer',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 30,
              boxShadow: isConnected
                ? `0 0 24px ${primaryHex}80`
                : '0 4px 12px rgba(0,0,0,0.12)',
              transition: 'transform 0.25s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.25s ease',
              outline: 'none',
            }}
          >
            {isBusy ? (
              <LoadingOutlined spin />
            ) : isError ? (
              <ExclamationCircleOutlined />
            ) : (
              <PoweroffOutlined />
            )}
          </button>
        </Tooltip>

        <div style={{ minWidth: 0, flex: 1 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              flexWrap: 'nowrap',
              whiteSpace: 'nowrap',
            }}
          >
            <span
              style={{
                fontSize: 20,
                fontWeight: 700,
                color: isDark ? '#f8fafc' : '#0f172a',
                flexShrink: 0,
              }}
            >
              {statusMeta.btnText}
            </span>
            <Badge status={statusMeta.badge} text={statusMeta.label} />
          </div>

          <div
            style={{
              marginTop: 10,
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              flexWrap: 'nowrap',
              overflow: 'hidden',
            }}
          >
            {activeNode ? (
              <>
                <Tag
                  color="processing"
                  icon={<ThunderboltFilled />}
                  style={{ margin: 0, maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis' }}
                >
                  当前节点：{activeNode.name}
                </Tag>
                <Tag style={{ margin: 0, flexShrink: 0 }}>
                  {activeNode.protocol.toUpperCase()}
                </Tag>
                <Tag style={{ margin: 0, flexShrink: 0 }}>
                  {activeNode.server}:{activeNode.port}
                </Tag>
                {activeNode.realDelay > 0 && (
                  <Tag
                    style={{ margin: 0, flexShrink: 0 }}
                    color={activeNode.realDelay < 200 ? 'success' : 'warning'}
                  >
                    {activeNode.realDelay} ms
                  </Tag>
                )}
              </>
            ) : (
              <Tag color="warning">尚未选择活动节点</Tag>
            )}
          </div>
        </div>
      </div>

      <div style={{ textAlign: 'right', flexShrink: 0, whiteSpace: 'nowrap' }}>
        <div
          style={{
            fontSize: 12,
            color: isDark ? '#94a3b8' : '#64748b',
            marginBottom: 8,
          }}
        >
          本地监听端点 ({appConfig.coreType.toUpperCase()})
        </div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <Tag color={isConnected ? 'geekblue' : 'default'} style={{ margin: 0 }}>
            SOCKS5: {appConfig.allowLan ? '0.0.0.0' : '127.0.0.1'}:
            {appConfig.socksPort}
          </Tag>
          <Tag color={isConnected ? 'cyan' : 'default'} style={{ margin: 0 }}>
            HTTP: {appConfig.allowLan ? '0.0.0.0' : '127.0.0.1'}:
            {appConfig.httpPort}
          </Tag>
        </div>
      </div>
    </div>
  );
};
