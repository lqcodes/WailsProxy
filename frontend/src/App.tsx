import React, { useEffect, useState } from 'react';
import {
  Badge,
  ConfigProvider,
  Layout,
  Menu,
  Space,
  Tag,
  Tooltip,
  theme as antdTheme,
} from 'antd';
import {
  ApartmentOutlined,
  CloudServerOutlined,
  CodeOutlined,
  DashboardOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import {
  EventsOn,
  Quit,
  WindowMinimise,
  WindowToggleMaximise,
  config,
} from './api';
import { DependencyGate } from './components/DependencyGate';
import { ThemeSwitcher } from './components/ThemeSwitcher';
import { useConfigStore } from './stores/configStore';
import { useNodeStore } from './stores/nodeStore';
import { useThemeStore, COLOR_MAP } from './stores/themeStore';
import { useTrafficStore } from './stores/trafficStore';
import { Dashboard } from './views/Dashboard';
import { Servers } from './views/Servers';
import { Rules } from './views/Rules';
import { Settings } from './views/Settings';
import { Logs } from './views/Logs';

const { Sider, Header, Content } = Layout;

type ViewKey = 'dashboard' | 'servers' | 'rules' | 'settings' | 'logs';

export const App: React.FC = () => {
  const [activeView, setActiveView] = useState<ViewKey>('dashboard');
  const [depsReady, setDepsReady] = useState<boolean>(true);
  const [depModalOpen, setDepModalOpen] = useState<boolean>(false);

  const isDark = useThemeStore((s) => s.isDark);
  const themeColor = useThemeStore((s) => s.color);
  const themeMode = useThemeStore((s) => s.mode);
  const syncFromConfig = useThemeStore((s) => s.syncFromConfig);

  const status = useConfigStore((s) => s.status);
  const loadConfig = useConfigStore((s) => s.loadConfig);
  const setStatus = useConfigStore((s) => s.setStatus);

  const fetchAllNodes = useNodeStore((s) => s.fetchAll);
  const applyPingResult = useNodeStore((s) => s.applyPingResult);

  const pushTrafficStats = useTrafficStore((s) => s.pushTrafficStats);
  const appendLog = useTrafficStore((s) => s.appendLog);
  const loadInitialLogs = useTrafficStore((s) => s.loadInitialLogs);

  useEffect(() => {
    void loadConfig();
    void fetchAllNodes();
    void loadInitialLogs();

    const offTraffic = EventsOn('traffic:stats', (stats: config.TrafficStats) => {
      pushTrafficStats(stats);
    });
    const offStatus = EventsOn(
      'connection:status',
      (st: config.ConnectionStatus) => {
        setStatus(st);
      }
    );
    const offLog = EventsOn('core:log', (line: string) => {
      appendLog(line);
    });
    const offPing = EventsOn('ping:result', (res: config.PingResultEvent) => {
      applyPingResult(res);
    });
    const offSub = EventsOn('subscription:updated', () => {
      void fetchAllNodes();
    });

    return () => {
      offTraffic();
      offStatus();
      offLog();
      offPing();
      offSub();
    };
  }, [
    loadConfig,
    fetchAllNodes,
    loadInitialLogs,
    pushTrafficStats,
    setStatus,
    appendLog,
    applyPingResult,
  ]);

  // 监听系统暗黑模式变化（当 themeMode === 'auto' 时自动响应）
  useEffect(() => {
    if (themeMode !== 'auto' || !window.matchMedia) return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => syncFromConfig('auto', themeColor);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [themeMode, themeColor, syncFromConfig]);

  const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';

  // 同步全局 CSS 变量与 data-theme 属性，确保所有 Portal 弹窗、抽屉、Toast 消息及确认框实时适配当前主题与主题色
  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', isDark ? 'dark' : 'light');
    root.style.setProperty('--primary-color', primaryHex);
    root.style.setProperty('--bg-color', isDark ? '#0f172a' : '#f5f7fa');
    root.style.setProperty('--card-bg', isDark ? '#1e293b' : '#ffffff');
    root.style.setProperty('--modal-bg', isDark ? '#1e293b' : '#ffffff');
    root.style.setProperty('--header-bg', isDark ? '#0b1120' : '#f8fafc');
    root.style.setProperty('--text-color', isDark ? '#f8fafc' : '#0f172a');
    root.style.setProperty('--muted-color', isDark ? '#94a3b8' : '#64748b');
    root.style.setProperty('--border-color', isDark ? '#334155' : '#e2e8f0');
  }, [isDark, primaryHex]);

  const renderContent = () => {
    switch (activeView) {
      case 'dashboard':
        return <Dashboard />;
      case 'servers':
        return <Servers />;
      case 'rules':
        return <Rules />;
      case 'settings':
        return <Settings />;
      case 'logs':
        return <Logs />;
    }
  };

  // 独立最顶部一行：macOS 风格红黄绿三色窗口控制栏
  const renderMacTopTitleBar = () => (
    <div
      onDoubleClick={() => WindowToggleMaximise()}
      style={{
        height: 36,
        minHeight: 36,
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '0 16px',
        background: isDark ? '#070c18' : '#eef2f6',
        borderBottom: `1px solid ${isDark ? '#1e293b' : '#dce3ec'}`,
        // @ts-expect-error Wails custom CSS property for window dragging
        '--wails-draggable': 'drag',
        flexShrink: 0,
        zIndex: 100,
      }}
    >
      {/* 左上角 macOS 三色窗口控制按钮（关闭 / 最小化 / 放大还原） */}
      <div
        className="mac-traffic-lights"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          // @ts-expect-error Wails custom CSS property
          '--wails-draggable': 'no-drag',
        }}
      >
        <Tooltip title="关闭窗口 (安全退出并还原代理)">
          <button
            type="button"
            onClick={() => Quit()}
            aria-label="Close"
            style={{
              width: 13,
              height: 13,
              borderRadius: '50%',
              background: '#ff5f57',
              border: '1px solid #e0443e',
              padding: 0,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#4d0000',
              fontSize: 10,
              fontWeight: 800,
              lineHeight: 1,
            }}
          >
            <span className="mac-traffic-icon" style={{ opacity: 0, transition: 'opacity 0.15s' }}>
              ×
            </span>
          </button>
        </Tooltip>

        <Tooltip title="最小化窗口">
          <button
            type="button"
            onClick={() => WindowMinimise()}
            aria-label="Minimize"
            style={{
              width: 13,
              height: 13,
              borderRadius: '50%',
              background: '#febc2e',
              border: '1px solid #dea123',
              padding: 0,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#5a3000',
              fontSize: 10,
              fontWeight: 800,
              lineHeight: 1,
            }}
          >
            <span className="mac-traffic-icon" style={{ opacity: 0, transition: 'opacity 0.15s' }}>
              −
            </span>
          </button>
        </Tooltip>

        <Tooltip title="最大化 / 还原窗口">
          <button
            type="button"
            onClick={() => WindowToggleMaximise()}
            aria-label="Zoom"
            style={{
              width: 13,
              height: 13,
              borderRadius: '50%',
              background: '#28c840',
              border: '1px solid #1aab29',
              padding: 0,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#003d00',
              fontSize: 10,
              fontWeight: 800,
              lineHeight: 1,
            }}
          >
            <span className="mac-traffic-icon" style={{ opacity: 0, transition: 'opacity 0.15s' }}>
              +
            </span>
          </button>
        </Tooltip>
      </div>

      {/* 居中窗口标题 */}
      <div
        style={{
          fontSize: 12,
          fontWeight: 600,
          color: isDark ? '#94a3b8' : '#475569',
          letterSpacing: 0.3,
          pointerEvents: 'none',
        }}
      >
        WailsProxy — 跨平台代理控制台
      </div>

      {/* 右侧占位保持标题居中 */}
      <div style={{ width: 55 }} />
    </div>
  );

  return (
    <ConfigProvider
      theme={{
        algorithm: isDark
          ? antdTheme.darkAlgorithm
          : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: primaryHex,
          colorInfo: primaryHex,
          borderRadius: 10,
          colorBgElevated: isDark ? '#1e293b' : '#ffffff',
          colorBgContainer: isDark ? '#0f172a' : '#ffffff',
          colorText: isDark ? '#f8fafc' : '#0f172a',
          colorTextSecondary: isDark ? '#94a3b8' : '#475569',
          colorBorder: isDark ? '#334155' : '#e2e8f0',
        },
        components: {
          Modal: {
            contentBg: isDark ? '#1e293b' : '#ffffff',
            headerBg: isDark ? '#1e293b' : '#ffffff',
            titleColor: isDark ? '#f8fafc' : '#0f172a',
          },
          Drawer: {
            colorBgElevated: isDark ? '#1e293b' : '#ffffff',
          },
          Message: {
            contentBg: isDark ? '#1e293b' : '#ffffff',
          },
          Popover: {
            colorBgElevated: isDark ? '#1e293b' : '#ffffff',
          },
          Dropdown: {
            colorBgElevated: isDark ? '#1e293b' : '#ffffff',
          },
          Select: {
            colorBgElevated: isDark ? '#1e293b' : '#ffffff',
          },
        },
      }}
    >
      <div
        style={{
          height: '100vh',
          minWidth: 1020,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          borderRadius: 16,
          background: isDark ? '#0f172a' : '#f5f7fa',
          border: `1px solid ${
            isDark ? 'rgba(148, 163, 184, 0.22)' : 'rgba(15, 23, 42, 0.14)'
          }`,
        }}
      >
        {/* 窗口最顶部独立一行：macOS 红黄绿缩小/放大/关闭控制栏 */}
        {renderMacTopTitleBar()}

        <Layout style={{ flex: 1, minHeight: 0, minWidth: 1020 }}>
          <Sider
            width={215}
            style={{
              background: isDark ? '#0b1120' : '#ffffff',
              borderRight: `1px solid ${isDark ? '#1e293b' : '#e2e8f0'}`,
              flexShrink: 0,
            }}
          >
            <div
              style={{
                height: 58,
                display: 'flex',
                alignItems: 'center',
                padding: '0 20px',
                gap: 10,
                borderBottom: `1px solid ${isDark ? '#1e293b' : '#f1f5f9'}`,
              }}
            >
              <img
                src="/appicon.png"
                alt="WailsProxy"
                style={{
                  width: 34,
                  height: 34,
                  borderRadius: 9,
                  flexShrink: 0,
                  boxShadow: `0 4px 12px ${primaryHex}55`,
                }}
              />
              <div>
                <div
                  style={{
                    fontWeight: 700,
                    fontSize: 16,
                    letterSpacing: 0.3,
                    color: isDark ? '#f8fafc' : '#0f172a',
                    lineHeight: 1.2,
                  }}
                >
                  WailsProxy
                </div>
                <div
                  style={{
                    fontSize: 11,
                    color: isDark ? '#64748b' : '#94a3b8',
                  }}
                >
                  Core v2.0 GUI
                </div>
              </div>
            </div>

            <Menu
              mode="inline"
              selectedKeys={[activeView]}
              onClick={(e) => setActiveView(e.key as ViewKey)}
              style={{
                borderRight: 0,
                background: 'transparent',
                padding: '10px 8px',
              }}
              items={[
                {
                  key: 'dashboard',
                  icon: <DashboardOutlined />,
                  label: '实时仪表盘',
                },
                {
                  key: 'servers',
                  icon: <CloudServerOutlined />,
                  label: '节点与订阅',
                },
                {
                  key: 'rules',
                  icon: <ApartmentOutlined />,
                  label: '分流与规则',
                },
                {
                  key: 'logs',
                  icon: <CodeOutlined />,
                  label: '内核实时日志',
                },
                {
                  key: 'settings',
                  icon: <SettingOutlined />,
                  label: '系统与DNS设置',
                },
              ]}
            />
          </Sider>

          <Layout
            style={{
              background: isDark ? '#0f172a' : '#f5f7fa',
              minWidth: 805,
            }}
          >
            <Header
              style={{
                height: 58,
                padding: '0 24px',
                background: isDark ? 'rgba(15, 23, 42, 0.85)' : '#ffffff',
                borderBottom: `1px solid ${isDark ? '#1e293b' : '#e2e8f0'}`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                flexShrink: 0,
              }}
            >
              <Space size={10}>
                <Badge
                  status={
                    status === 'connected'
                      ? 'success'
                      : status === 'connecting'
                      ? 'processing'
                      : status === 'error'
                      ? 'error'
                      : 'default'
                  }
                />
                <Tag color={status === 'connected' ? 'success' : 'default'}>
                  {status === 'connected'
                    ? '已连接保护中'
                    : status === 'connecting'
                    ? '正在启动内核'
                    : status === 'error'
                    ? '连接异常'
                    : '待机未连接'}
                </Tag>
              </Space>

              <Space size={12}>
                <Tag
                  color={depsReady ? 'processing' : 'error'}
                  style={{ cursor: 'pointer' }}
                  onClick={() => setDepModalOpen(true)}
                >
                  {depsReady ? '内核依赖已就绪 (点击自检)' : '依赖未就绪 (点击修复)'}
                </Tag>
                <ThemeSwitcher />
              </Space>
            </Header>

            <Content
              style={{
                padding: '18px 24px',
                overflowY: 'auto',
                height: 'calc(100vh - 94px)',
              }}
            >
              {renderContent()}
            </Content>
          </Layout>
        </Layout>
      </div>

      {/* 独立内核依赖检测弹窗（Portal 渲染，不影响主界面布局与原有样式，手动关闭） */}
      <DependencyGate
        open={depModalOpen || !depsReady}
        onClose={() => {
          if (depsReady) {
            setDepModalOpen(false);
          }
        }}
        onStatusChange={(ready) => {
          setDepsReady(ready);
          if (!ready) {
            setDepModalOpen(true);
          }
        }}
      />
    </ConfigProvider>
  );
};

export default App;
