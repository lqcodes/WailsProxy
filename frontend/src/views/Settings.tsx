import React, { useEffect } from 'react';
import {
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Switch,
  message,
} from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useConfigStore } from '../stores/configStore';

export const Settings: React.FC = () => {
  const appConfig = useConfigStore((s) => s.appConfig);
  const updateConfig = useConfigStore((s) => s.updateConfig);
  const [form] = Form.useForm();

  useEffect(() => {
    form.setFieldsValue(appConfig);
  }, [appConfig, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    if (values.socksPort === values.httpPort) {
      message.error('Socks5 端口与 HTTP 端口不能相同');
      return;
    }
    await updateConfig(values);
    message.success('全局配置已原子保存并生效');
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Form form={form} layout="vertical" initialValues={appConfig}>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small" title="本地代理监听与内核设置">
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item
                    name="socksPort"
                    label="本地 SOCKS5 端口"
                    rules={[{ required: true }]}
                  >
                    <InputNumber min={1024} max={65535} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="httpPort"
                    label="本地 HTTP 端口"
                    rules={[{ required: true }]}
                  >
                    <InputNumber min={1024} max={65535} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="coreType" label="开源代理内核">
                    <Select
                      options={[
                        { label: 'Sing-box (推荐 / 原生 TUN)', value: 'sing-box' },
                        { label: 'Xray-core', value: 'xray' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="allowLan"
                    label="允许局域网共享 (监听 0.0.0.0)"
                    valuePropName="checked"
                  >
                    <Switch />
                  </Form.Item>
                </Col>
              </Row>
            </Card>
          </Col>

          <Col xs={24} md={12}>
            <Card size="small" title="自定义 DNS 与防污染配置">
              <Form.Item
                name="directDns"
                label="国内直连 DNS (Direct DNS)"
                rules={[{ required: true }]}
              >
                <Input placeholder="223.5.5.5 或 119.29.29.29" />
              </Form.Item>

              <Form.Item
                name="remoteDns"
                label="远程代理 DNS (Remote DNS / DoH)"
                rules={[{ required: true }]}
              >
                <Input placeholder="https://1.1.1.1/dns-query 或 8.8.8.8" />
              </Form.Item>

              <Form.Item
                name="enableDoh"
                label="启用 DNS over HTTPS (DoH) 加密防污染"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Card>
          </Col>

          <Col xs={24} md={12}>
            <Card size="small" title="节点测速参数配置">
              <Form.Item
                name="latencyTestUrl"
                label="HTTP 204 真实延迟探测地址"
                rules={[{ required: true }]}
              >
                <Input placeholder="http://cp.cloudflare.com/generate_204" />
              </Form.Item>

              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="latencyTimeout" label="测速超时 (毫秒)">
                    <InputNumber min={500} max={15000} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="latencyMaxConc" label="最大并发测速数">
                    <InputNumber min={1} max={100} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>
          </Col>

          <Col xs={24} md={12}>
            <Card size="small" title="桌面托盘与系统行为">
              <Form.Item
                name="closeToTray"
                label="点击关闭按钮时最小化到系统托盘 (防误关断网)"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>

              <Form.Item
                name="autoStart"
                label="开机自动启动 WailsProxy"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>

              <Button
                type="primary"
                size="large"
                icon={<SaveOutlined />}
                block
                onClick={handleSave}
              >
                保存全部设置
              </Button>
            </Card>
          </Col>
        </Row>
      </Form>
    </Space>
  );
};
