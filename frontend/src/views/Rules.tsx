import React, { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Input,
  Row,
  Space,
  Tag,
  message,
} from 'antd';
import {
  ReloadOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  DEFAULT_BLOCK_RULES,
  DEFAULT_DIRECT_RULES,
  DEFAULT_PROXY_RULES,
  useConfigStore,
} from '../stores/configStore';

export const Rules: React.FC = () => {
  const appConfig = useConfigStore((s) => s.appConfig);
  const updateConfig = useConfigStore((s) => s.updateConfig);

  const [directText, setDirectText] = useState('');
  const [proxyText, setProxyText] = useState('');
  const [blockText, setBlockText] = useState('');

  useEffect(() => {
    setDirectText((appConfig.customDirectRules || []).join('\n'));
    setProxyText((appConfig.customProxyRules || []).join('\n'));
    setBlockText((appConfig.customBlockRules || []).join('\n'));
  }, [
    appConfig.customDirectRules,
    appConfig.customProxyRules,
    appConfig.customBlockRules,
  ]);

  const parseLines = (txt: string) =>
    txt
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean);

  const handleSaveRules = async () => {
    await updateConfig({
      customDirectRules: parseLines(directText),
      customProxyRules: parseLines(proxyText),
      customBlockRules: parseLines(blockText),
    });
    message.success('自定义分流规则已保存并热重载生效');
  };

  const handleRestoreDefaults = async () => {
    setDirectText(DEFAULT_DIRECT_RULES.join('\n'));
    setProxyText(DEFAULT_PROXY_RULES.join('\n'));
    setBlockText(DEFAULT_BLOCK_RULES.join('\n'));
    await updateConfig({
      customDirectRules: DEFAULT_DIRECT_RULES,
      customProxyRules: DEFAULT_PROXY_RULES,
      customBlockRules: DEFAULT_BLOCK_RULES,
    });
    message.success('已恢复默认常用分流规则并热重载生效');
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="success"
        showIcon
        icon={<SafetyCertificateOutlined />}
        message="三重防 DNS 泄漏保护已开启 (Zero DNS Leak Protection)"
        description="① 海外与未识别域名强制通过代理隧道走加密 DoH 解析；② 国内 DNS (223.5.5.5) 严格限定在大陆白名单域名且开启 skipFallback 禁止跨区回退；③ 远程代理请求采用域名直传 (AsIs) + 拦截本地 UDP/443 QUIC 绕行，彻底杜绝运营商 DNS 抢答与泄漏。"
      />

      <Card
        size="small"
        title="常用与自定义分流规则 (每行一条域名后缀或 IP CIDR)"
        extra={
          <Space size={8}>
            <Button icon={<ReloadOutlined />} onClick={handleRestoreDefaults}>
              恢复默认常用规则
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={handleSaveRules}
            >
              保存并热重载规则
            </Button>
          </Space>
        }
      >
        <Row gutter={[14, 14]}>
          <Col xs={24} md={8}>
            <div style={{ marginBottom: 6 }}>
              <Tag color="green">直连规则 (Direct)</Tag>
              <span style={{ fontSize: 12, opacity: 0.7 }}>
                强制不走代理的域名或网段
              </span>
            </div>
            <Input.TextArea
              rows={12}
              value={directText}
              onChange={(e) => setDirectText(e.target.value)}
              placeholder={'internal.corp.com\n192.168.0.0/16\ncn'}
            />
          </Col>

          <Col xs={24} md={8}>
            <div style={{ marginBottom: 6 }}>
              <Tag color="geekblue">代理规则 (Proxy)</Tag>
              <span style={{ fontSize: 12, opacity: 0.7 }}>
                强制走代理节点的域名或网段
              </span>
            </div>
            <Input.TextArea
              rows={12}
              value={proxyText}
              onChange={(e) => setProxyText(e.target.value)}
              placeholder={'openai.com\ngithub.com\ngoogle.com'}
            />
          </Col>

          <Col xs={24} md={8}>
            <div style={{ marginBottom: 6 }}>
              <Tag color="red">拦截规则 (Block)</Tag>
              <span style={{ fontSize: 12, opacity: 0.7 }}>
                广告或恶意追踪域名黑名单
              </span>
            </div>
            <Input.TextArea
              rows={12}
              value={blockText}
              onChange={(e) => setBlockText(e.target.value)}
              placeholder={'ads.example.com\ndoubleclick.net'}
            />
          </Col>
        </Row>
      </Card>
    </Space>
  );
};
