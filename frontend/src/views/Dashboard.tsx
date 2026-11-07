import React from 'react';
import {
  Button,
  Card,
  Col,
  Row,
  Space,
  Statistic,
  Tag,
  Tooltip,
  message,
} from 'antd';
import {
  ApartmentOutlined,
  ApiOutlined,
  ArrowDownOutlined,
  ArrowUpOutlined,
  ClusterOutlined,
  CodeOutlined,
  CompassOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  GlobalOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { ConnectButton } from '../components/ConnectButton';
import { TrafficChart } from '../components/TrafficChart';
import { useConfigStore } from '../stores/configStore';
import { useThemeStore, COLOR_MAP } from '../stores/themeStore';
import { useTrafficStore } from '../stores/trafficStore';
import { ClipboardSetText, GetTerminalProxyCommand } from '../api';
import { formatSpeed, formatTraffic } from '../utils/format';

interface ModeOption {
  value: string;
  title: string;
  subtitle: string;
  icon: React.ReactNode;
}

const PROXY_MODE_OPTIONS: ModeOption[] = [
  {
    value: 'system',
    title: '系统代理 (System)',
    subtitle: '自动配置 macOS / Windows 浏览器全局代理',
    icon: <DesktopOutlined />,
  },
  {
    value: 'manual',
    title: '纯代理监听 (Manual)',
    subtitle: '仅开放本地 SOCKS5 / HTTP 端口，不改系统设置',
    icon: <ApiOutlined />,
  },
  {
    value: 'tun',
    title: 'TUN 虚拟网卡 (TUN)',
    subtitle: '透明接管底层全局网络流量与终端应用',
    icon: <ThunderboltOutlined />,
  },
];

const ROUTING_MODE_OPTIONS: ModeOption[] = [
  {
    value: 'rule',
    title: '规则分流 (Rule)',
    subtitle: '智能分流：国内域名/IP 直连，Google 等外网走代理',
    icon: <ApartmentOutlined />,
  },
  {
    value: 'global',
    title: '全局代理 (Global)',
    subtitle: '所有外网流量统一通过当前活动节点转发',
    icon: <CompassOutlined />,
  },
  {
    value: 'direct',
    title: '直接连接 (Direct)',
    subtitle: '全部流量直接连接，不经过代理节点',
    icon: <DisconnectOutlined />,
  },
];

export const Dashboard: React.FC = () => {
  const appConfig = useConfigStore((s) => s.appConfig);
  const changeProxyMode = useConfigStore((s) => s.changeProxyMode);
  const changeRoutingMode = useConfigStore((s) => s.changeRoutingMode);
  const currentTraffic = useTrafficStore((s) => s.current);
  const isDark = useThemeStore((s) => s.isDark);
  const themeColor = useThemeStore((s) => s.color);
  const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';

  const handleCopyTerminalCmd = async () => {
    const cmd = await GetTerminalProxyCommand('zsh');
    await ClipboardSetText(cmd);
    message.success('已复制终端代理环境变量命令到剪贴板');
  };

  const renderModeRow = (
    rowLabel: string,
    rowBadge: string,
    currentValue: string,
    options: ModeOption[],
    onSelect: (val: string) => Promise<void>,
    extraAction?: React.ReactNode
  ) => (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        flexWrap: 'nowrap',
      }}
    >
      {/* 左侧行标题 */}
      <div
        style={{
          width: 112,
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 3,
        }}
      >
        <span
          style={{
            fontSize: 14,
            fontWeight: 600,
            color: isDark ? '#f1f5f9' : '#1e293b',
          }}
        >
          {rowLabel}
        </span>
        <Tag
          color="default"
          style={{
            margin: 0,
            width: 'fit-content',
            fontSize: 11,
            lineHeight: '18px',
            padding: '0 6px',
          }}
        >
          {rowBadge}
        </Tag>
      </div>

      {/* 中间三列等宽卡片式切换按钮 */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
          gap: 10,
          flex: 1,
        }}
      >
        {options.map((opt) => {
          const active = currentValue === opt.value;
          return (
            <button
              key={opt.value}
              type="button"
              onClick={() => void onSelect(opt.value)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '10px 14px',
                borderRadius: 10,
                cursor: 'pointer',
                textAlign: 'left',
                border: active
                  ? `1.5px solid ${primaryHex}`
                  : `1px solid ${isDark ? '#334155' : '#e2e8f0'}`,
                background: active
                  ? isDark
                    ? `${primaryHex}24`
                    : `${primaryHex}12`
                  : isDark
                  ? 'rgba(15, 23, 42, 0.55)'
                  : '#f8fafc',
                boxShadow: active ? `0 4px 14px ${primaryHex}26` : 'none',
                transition: 'all 0.2s ease',
                outline: 'none',
              }}
            >
              <div
                style={{
                  width: 34,
                  height: 34,
                  borderRadius: 8,
                  flexShrink: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 17,
                  background: active
                    ? primaryHex
                    : isDark
                    ? '#1e293b'
                    : '#e2e8f0',
                  color: active
                    ? '#ffffff'
                    : isDark
                    ? '#94a3b8'
                    : '#475569',
                  transition: 'all 0.2s ease',
                }}
              >
                {opt.icon}
              </div>
              <div style={{ minWidth: 0, flex: 1 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: 600,
                    color: active
                      ? isDark
                        ? '#ffffff'
                        : primaryHex
                      : isDark
                      ? '#e2e8f0'
                      : '#1e293b',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {opt.title}
                </div>
                <div
                  style={{
                    fontSize: 11,
                    color: isDark ? '#94a3b8' : '#64748b',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    marginTop: 2,
                  }}
                >
                  {opt.subtitle}
                </div>
              </div>
            </button>
          );
        })}
      </div>

      {/* 右侧快捷操作区（对齐宽度） */}
      <div style={{ width: 148, flexShrink: 0, display: 'flex', justifyContent: 'flex-end' }}>
        {extraAction}
      </div>
    </div>
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* 主控一键连接/断开面板 */}
      <ConnectButton />

      {/* 两行独立显示：第 1 行「连接模式」+ 第 2 行「路由分流」 */}
      <Card
        size="small"
        styles={{
          body: {
            padding: '16px 20px',
            display: 'flex',
            flexDirection: 'column',
            gap: 14,
          },
        }}
      >
        {/* 第一行：连接模式 */}
        {renderModeRow(
          '连接模式',
          'Proxy Mode',
          appConfig.proxyMode,
          PROXY_MODE_OPTIONS,
          async (val) => {
            await changeProxyMode(val as 'system' | 'manual' | 'tun');
            message.success(`已切换连接模式为：${val.toUpperCase()}`);
          },
          <Tooltip title="复制 export http_proxy=... 终端环境变量命令">
            <Button
              icon={<CodeOutlined />}
              onClick={handleCopyTerminalCmd}
              style={{ borderRadius: 8, width: '100%' }}
            >
              终端代理命令
            </Button>
          </Tooltip>
        )}

        <div
          style={{
            height: 1,
            background: isDark ? '#1e293b' : '#f1f5f9',
            width: '100%',
          }}
        />

        {/* 第二行：路由分流 */}
        {renderModeRow(
          '路由分流',
          'Routing Mode',
          appConfig.routingMode,
          ROUTING_MODE_OPTIONS,
          async (val) => {
            await changeRoutingMode(val as 'rule' | 'global' | 'direct');
            message.success(`路由分流模式已热重载为：${val.toUpperCase()}`);
          },
          <Tag
            color="processing"
            style={{
              margin: 0,
              padding: '5px 10px',
              borderRadius: 8,
              width: '100%',
              textAlign: 'center',
              fontSize: 12,
            }}
          >
            支持连接中秒级热切
          </Tag>
        )}
      </Card>

      {/* 实时速率与累计流量 4 卡片 */}
      <Row gutter={[14, 14]}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="实时下载速率"
              value={formatSpeed(currentTraffic.downloadSpeed)}
              prefix={<ArrowDownOutlined style={{ color: '#1677ff' }} />}
            />
            <div style={{ marginTop: 6, fontSize: 12, opacity: 0.7 }}>
              累计下载：{formatTraffic(currentTraffic.totalDownload)}
            </div>
          </Card>
        </Col>

        <Col span={6}>
          <Card size="small">
            <Statistic
              title="实时上传速率"
              value={formatSpeed(currentTraffic.uploadSpeed)}
              prefix={<ArrowUpOutlined style={{ color: '#10b981' }} />}
            />
            <div style={{ marginTop: 6, fontSize: 12, opacity: 0.7 }}>
              累计上传：{formatTraffic(currentTraffic.totalUpload)}
            </div>
          </Card>
        </Col>

        <Col span={6}>
          <Card size="small">
            <Statistic
              title="活跃并发连接"
              value={currentTraffic.activeConns}
              suffix="条"
              prefix={<ClusterOutlined style={{ color: '#722ed1' }} />}
            />
            <div style={{ marginTop: 6, fontSize: 12, opacity: 0.7 }}>
              内核引擎：{appConfig.coreType.toUpperCase()}
            </div>
          </Card>
        </Col>

        <Col span={6}>
          <Card size="small">
            <Statistic
              title="防污染 DNS 引擎"
              value={appConfig.enableDoh ? 'DoH 加密' : '标准 DNS'}
              prefix={
                appConfig.enableDoh ? (
                  <SafetyCertificateOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <GlobalOutlined />
                )
              }
            />
            <div
              style={{
                marginTop: 6,
                fontSize: 12,
                opacity: 0.7,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              直连 {appConfig.directDns} | 远程 {appConfig.remoteDns}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 60 秒实时平滑流量波形图 */}
      <Card size="small">
        <TrafficChart />
      </Card>
    </Space>
  );
};
