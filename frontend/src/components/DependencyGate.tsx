import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  List,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  CheckCircleFilled,
  DownloadOutlined,
  ReloadOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import {
  CheckDependencies,
  InstallDependencies,
  config,
} from '../api';
import { COLOR_MAP, useThemeStore } from '../stores/themeStore';

const { Text } = Typography;

interface DependencyGateProps {
  open: boolean;
  onClose: () => void;
  onStatusChange?: (ready: boolean) => void;
}

export const DependencyGate: React.FC<DependencyGateProps> = ({
  open,
  onClose,
  onStatusChange,
}) => {
  const [loading, setLoading] = useState<boolean>(false);
  const [installing, setInstalling] = useState<boolean>(false);
  const [installingId, setInstallingId] = useState<string | null>(null);
  const [result, setResult] = useState<config.DependencyCheckResult | null>(
    null
  );

  const isDark = useThemeStore((s) => s.isDark);
  const themeColor = useThemeStore((s) => s.color);
  const primaryHex = COLOR_MAP[themeColor]?.hex || '#1677ff';

  const runCheck = useCallback(
    async (silent = false, showToast = false) => {
      if (!silent) {
        setLoading(true);
      }
      try {
        // 显式检测时保留平滑的 600ms 扫描反馈动画，避免 1ms 瞬闪导致视觉突兀
        const [res] = await Promise.all([
          CheckDependencies(),
          silent
            ? Promise.resolve()
            : new Promise((resolve) => setTimeout(resolve, 600)),
        ]);
        setResult(res);
        onStatusChange?.(res.ready);
        if (showToast) {
          if (res.ready) {
            message.success('内核与分流规则依赖检测完成：全部就绪');
          } else {
            message.warning('检测完成：存在缺失组件，请点击对应项右侧「点击安装」');
          }
        }
        return res;
      } catch (err: unknown) {
        if (!silent) {
          message.error(
            `依赖检测异常: ${err instanceof Error ? err.message : String(err)}`
          );
        }
        return null;
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [onStatusChange]
  );

  // 启动时后台静默检测并自动准备默认目录与内置内核
  useEffect(() => {
    void runCheck(true, false);
  }, [runCheck]);

  // 当弹窗打开时执行带平滑过渡的检测，且绝不自动关闭，交由用户手动关闭
  useEffect(() => {
    if (open) {
      void runCheck(false, false);
    }
  }, [open, runCheck]);

  const handleInstall = async (targetItemId?: string) => {
    setInstalling(true);
    if (targetItemId) {
      setInstallingId(targetItemId);
    }
    try {
      const [res] = await Promise.all([
        InstallDependencies(),
        new Promise((resolve) => setTimeout(resolve, 600)),
      ]);
      setResult(res);
      onStatusChange?.(res.ready);
      if (res.ready) {
        message.success('核心依赖已全部安装并加载就绪！确认无误后可点击下方按钮手动关闭');
      } else {
        message.warning(res.message);
      }
    } catch (err: unknown) {
      message.error(
        `自动安装依赖失败: ${err instanceof Error ? err.message : String(err)}`
      );
      await runCheck(false, false);
    } finally {
      setInstalling(false);
      setInstallingId(null);
    }
  };

  const canClose = Boolean(result?.ready);

  return (
    <Modal
      open={open}
      width={860}
      centered
      destroyOnClose={false}
      closable={canClose}
      maskClosable={false}
      keyboard={canClose}
      onCancel={() => {
        if (canClose) {
          onClose();
        } else {
          message.warning('请先完成所有必需依赖项的安装后再关闭窗口');
        }
      }}
      title={
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            paddingRight: canClose ? 28 : 0,
          }}
        >
          <Space size={10}>
            <SafetyCertificateOutlined
              style={{ color: primaryHex, fontSize: 20 }}
            />
            <span style={{ color: isDark ? '#f8fafc' : '#0f172a' }}>
              核心运行环境与开源内核依赖自检
            </span>
            {result?.platform && <Tag color="geekblue">{result.platform}</Tag>}
          </Space>
          {result && (
            <Tag
              icon={result.ready ? <CheckCircleFilled /> : undefined}
              color={result.ready ? 'success' : 'error'}
            >
              {result.ready ? '已就绪' : '存在未安装组件'}
            </Tag>
          )}
        </div>
      }
      footer={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Text type="secondary" style={{ fontSize: 12 }}>
            {canClose
              ? '系统已自动托管所有核心依赖，添加订阅即可直接使用（请手动关闭窗口）'
              : '提示：请点击缺失项右侧的「点击安装」完成自动部署'}
          </Text>
          <Space size={10}>
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={() => void runCheck(false, true)}
              disabled={installing}
            >
              重新检测
            </Button>

            {!canClose ? (
              <Button
                type="primary"
                icon={<DownloadOutlined />}
                loading={installing && !installingId}
                onClick={() => void handleInstall()}
              >
                一键全部安装
              </Button>
            ) : (
              <Button
                type="primary"
                icon={<RocketOutlined />}
                disabled={loading || installing}
                onClick={onClose}
              >
                手动关闭并继续使用
              </Button>
            )}
          </Space>
        </div>
      }
    >
      <Spin
        spinning={loading || installing}
        tip={
          installing
            ? '正在为您自动下载并部署缺失的核心依赖...'
            : '正在检测并加载代理内核 (Xray / Sing-box) 与分流规则库...'
        }
      >
        <Space
          direction="vertical"
          size={14}
          style={{ width: '100%', minHeight: 250 }}
        >
          {result && (
            <Alert
              type={result.ready ? 'success' : 'warning'}
              showIcon
              message={result.message}
            />
          )}

          <List
            bordered
            size="default"
            style={{
              background: isDark ? '#0f172a' : '#f8fafc',
              borderColor: isDark ? '#334155' : '#e2e8f0',
              borderRadius: 10,
            }}
            dataSource={result?.items || []}
            renderItem={(item) => (
              <List.Item
                style={{
                  borderBottomColor: isDark ? '#1e293b' : '#e2e8f0',
                  padding: '14px 18px',
                }}
                actions={[
                  item.installed ? (
                    <Tag
                      icon={
                        <CheckCircleFilled
                          style={{ color: '#52c41a', fontSize: 14 }}
                        />
                      }
                      color="success"
                      key="status"
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        padding: '4px 12px',
                        borderRadius: 6,
                        margin: 0,
                      }}
                    >
                      已就绪
                    </Tag>
                  ) : (
                    <Button
                      key="install"
                      type="primary"
                      size="small"
                      icon={<DownloadOutlined />}
                      loading={installingId === item.id || installing}
                      onClick={() => void handleInstall(item.id)}
                      style={{
                        borderRadius: 6,
                        fontWeight: 500,
                        padding: '0 12px',
                        height: 30,
                      }}
                    >
                      点击安装
                    </Button>
                  ),
                ]}
              >
                <List.Item.Meta
                  title={
                    <Space wrap size={8}>
                      <Text
                        strong
                        style={{
                          fontSize: 14,
                          color: isDark ? '#f8fafc' : '#0f172a',
                        }}
                      >
                        {item.name}
                      </Text>
                      {item.installed && item.version && (
                        <Tag
                          color="processing"
                          style={{ fontSize: 11, margin: 0, borderRadius: 4 }}
                        >
                          {item.version}
                        </Tag>
                      )}
                    </Space>
                  }
                  description={
                    <div
                      style={{
                        fontSize: 12.5,
                        marginTop: 4,
                        color: isDark ? '#94a3b8' : '#64748b',
                      }}
                    >
                      <div>{item.description}</div>
                      {!item.installed && item.error && (
                        <div style={{ marginTop: 4 }}>
                          <Text type="warning" style={{ fontSize: 12 }}>
                            {item.error}
                          </Text>
                        </div>
                      )}
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        </Space>
      </Spin>
    </Modal>
  );
};
