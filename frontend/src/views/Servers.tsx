import React, { useState } from 'react';
import {
  Badge,
  Button,
  Card,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  message,
} from 'antd';
import {
  ClearOutlined,
  CloudSyncOutlined,
  DeleteOutlined,
  FolderAddOutlined,
  FolderOpenOutlined,
  PlusOutlined,
  ReloadOutlined,
  SortAscendingOutlined,
  SwapOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { NodeTable } from '../components/NodeTable';
import { useNodeStore } from '../stores/nodeStore';
import { formatTimestamp } from '../utils/format';

export const Servers: React.FC = () => {
  const nodes = useNodeStore((s) => s.nodes);
  const subscriptions = useNodeStore((s) => s.subscriptions);
  const selectedGroupId = useNodeStore((s) => s.selectedGroupId);
  const searchQuery = useNodeStore((s) => s.searchQuery);
  const testingDelay = useNodeStore((s) => s.testingDelay);
  const refreshingSub = useNodeStore((s) => s.refreshingSub);

  const setSelectedGroup = useNodeStore((s) => s.setSelectedGroup);
  const setSearchQuery = useNodeStore((s) => s.setSearchQuery);
  const importShareLinks = useNodeStore((s) => s.importShareLinks);
  const pingGroupNodes = useNodeStore((s) => s.pingGroupNodes);
  const sortByDelay = useNodeStore((s) => s.sortByDelay);
  const cleanInvalidNodes = useNodeStore((s) => s.cleanInvalidNodes);
  const saveSubscription = useNodeStore((s) => s.saveSubscription);
  const refreshSubscription = useNodeStore((s) => s.refreshSubscription);
  const refreshAllSubscriptions = useNodeStore((s) => s.refreshAllSubscriptions);
  const removeSubscription = useNodeStore((s) => s.removeSubscription);
  const moveNodesToGroup = useNodeStore((s) => s.moveNodesToGroup);

  const [importModalOpen, setImportModalOpen] = useState(false);
  const [importTargetGroupId, setImportTargetGroupId] = useState<string>('default');
  const [rawLinksText, setRawLinksText] = useState('');
  const [subDrawerOpen, setSubDrawerOpen] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [subForm] = Form.useForm();

  // 按当前选中分组与搜索关键字过滤节点
  const filteredNodes = nodes.filter((n) => {
    const matchGroup =
      selectedGroupId === 'all' || n.subId === selectedGroupId;
    if (!matchGroup) return false;
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      n.name.toLowerCase().includes(q) ||
      n.server.toLowerCase().includes(q) ||
      n.protocol.toLowerCase().includes(q) ||
      (n.groupName || '').toLowerCase().includes(q)
    );
  });

  const handleImportSubmit = async () => {
    if (!rawLinksText.trim()) {
      message.warning('请粘贴节点分享链接 (支持 vmess/vless/trojan/ss/hy2/tuic/Base64/Clash YAML)');
      return;
    }
    try {
      const targetGroup =
        importTargetGroupId ||
        (selectedGroupId !== 'all' ? selectedGroupId : 'default');
      const count = await importShareLinks(rawLinksText, targetGroup);
      setSelectedGroup(targetGroup);
      message.success(`成功导入 ${count} 个节点`);
      setRawLinksText('');
      setImportModalOpen(false);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      message.error(`导入失败: ${msg}`);
    }
  };

  const handleAddSub = async () => {
    try {
      const values = await subForm.validateFields();
      const rawUrl = (values.url || '').trim();
      let subName = (values.name || '').trim();
      if (!subName && rawUrl) {
        try {
          const u = new URL(rawUrl);
          subName = `订阅 - ${u.hostname}`;
        } catch {
          subName = '远程订阅分组';
        }
      }
      if (!subName) {
        message.warning('请输入分组名称或订阅链接 URL');
        return;
      }
      message.loading({ content: '正在保存并拉取订阅节点...', key: 'add-sub' });
      await saveSubscription({
        id: '',
        name: subName,
        url: rawUrl,
        autoUpdate: Boolean(values.autoUpdate),
        updateInterval: Number(values.updateInterval || 12),
        lastUpdated: 0,
        nodeCount: 0,
      });
      subForm.resetFields();
      const latestSubs = useNodeStore.getState().subscriptions;
      const addedSub = latestSubs.find((s) => s.name === subName);
      if (addedSub) {
        setSelectedGroup(addedSub.id);
      }
      setSubDrawerOpen(false);
      message.success({
        content: `已成功添加订阅分组「${subName}」并同步节点`,
        key: 'add-sub',
      });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg && !msg.includes('errorFields')) {
        message.error({ content: `添加订阅失败: ${msg}`, key: 'add-sub' });
      }
    }
  };

  const tabItems = [
    {
      key: 'all',
      label: (
        <Space size={6}>
          <span>全部节点</span>
          <Badge count={nodes.length} showZero color="#1677ff" />
        </Space>
      ),
    },
    ...subscriptions.map((sub) => ({
      key: sub.id,
      label: (
        <Space size={6}>
          <span>{sub.name}</span>
          <Badge count={sub.nodeCount} showZero />
        </Space>
      ),
    })),
  ];

  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Card size="small">
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 10,
            marginBottom: 8,
          }}
        >
          <Input.Search
            allowClear
            placeholder="搜索节点名称 / 域名 / 协议..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: 260 }}
          />

          <Space wrap>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setImportTargetGroupId(
                  selectedGroupId !== 'all' ? selectedGroupId : 'default'
                );
                setImportModalOpen(true);
              }}
            >
              导入节点链接
            </Button>

            {selectedRowKeys.length > 0 && (
              <Dropdown
                trigger={['click']}
                menu={{
                  items: subscriptions.map((sub) => ({
                    key: sub.id,
                    icon: <FolderOpenOutlined />,
                    label: `移动至「${sub.name}」`,
                    onClick: async () => {
                      await moveNodesToGroup(
                        selectedRowKeys.map(String),
                        sub.id
                      );
                      message.success(
                        `已将选中的 ${selectedRowKeys.length} 个节点移动至分组「${sub.name}」`
                      );
                      setSelectedRowKeys([]);
                    },
                  })),
                }}
              >
                <Button icon={<SwapOutlined />}>
                  批量移动分组 ({selectedRowKeys.length})
                </Button>
              </Dropdown>
            )}

            <Button
              icon={<FolderAddOutlined />}
              onClick={() => setSubDrawerOpen(true)}
            >
              订阅分组管理
            </Button>

            <Button
              loading={refreshingSub}
              icon={<ReloadOutlined />}
              onClick={async () => {
                if (selectedGroupId === 'all') {
                  await refreshAllSubscriptions();
                  message.success('全部订阅已刷新');
                } else {
                  await refreshSubscription(selectedGroupId);
                  message.success('当前分组节点已刷新');
                }
              }}
            >
              刷新节点
            </Button>

            <Button
              loading={testingDelay}
              icon={<ThunderboltOutlined />}
              onClick={async () => {
                await pingGroupNodes(selectedGroupId);
                message.success('批量延迟测速完成');
              }}
            >
              延迟测试
            </Button>

            <Button
              icon={<SortAscendingOutlined />}
              onClick={async () => {
                await sortByDelay();
                message.success('已按延迟从低到高排序');
              }}
            >
              按延迟排序
            </Button>

            <Popconfirm
              title="确定清理所有测速超时 (-1) 的失效节点吗？"
              onConfirm={async () => {
                const count = await cleanInvalidNodes(selectedGroupId);
                message.info(`已清理 ${count} 个超时节点`);
              }}
            >
              <Button icon={<ClearOutlined />}>清理超时</Button>
            </Popconfirm>
          </Space>
        </div>

        <Tabs
          activeKey={selectedGroupId}
          onChange={setSelectedGroup}
          items={tabItems}
          style={{ marginBottom: 4 }}
        />

        <NodeTable
          dataSource={filteredNodes}
          selectedRowKeys={selectedRowKeys}
          onSelectChange={setSelectedRowKeys}
        />
      </Card>

      {/* 批量导入分享链接弹窗 */}
      <Modal
        title="批量导入节点 / 订阅内容"
        open={importModalOpen}
        onOk={handleImportSubmit}
        onCancel={() => setImportModalOpen(false)}
        okText="立即解析并导入"
        cancelText="取消"
        width={620}
      >
        <div
          style={{
            marginBottom: 12,
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}
        >
          <span style={{ fontSize: 13, fontWeight: 500 }}>导入到目标分组：</span>
          <Select
            style={{ width: 240 }}
            value={importTargetGroupId}
            onChange={(val) => setImportTargetGroupId(val)}
            options={subscriptions.map((s) => ({
              label: s.name,
              value: s.id,
            }))}
          />
        </div>
        <div style={{ marginBottom: 8, fontSize: 13, opacity: 0.8 }}>
          支持直接粘贴 <strong>远程订阅地址 (<code>https://...</code>)</strong>、多行分享链接（
          <code>vmess://</code>、<code>vless://</code>、<code>trojan://</code>、
          <code>ss://</code>、<code>hysteria2://</code>、<code>tuic://</code>）或
          Base64 / Clash YAML / Sing-box JSON 内容：
        </div>
        <Input.TextArea
          rows={8}
          value={rawLinksText}
          onChange={(e) => setRawLinksText(e.target.value)}
          placeholder={
            'https://qqg.cloudnote.cc:2096/...\n或 vless://uuid@server:443?security=reality...'
          }
        />
      </Modal>

      {/* 订阅与分组管理抽屉 */}
      <Drawer
        title="订阅源与分组管理"
        width={480}
        open={subDrawerOpen}
        onClose={() => setSubDrawerOpen(false)}
        extra={
          <Button
            size="small"
            type="primary"
            loading={refreshingSub}
            icon={<CloudSyncOutlined />}
            onClick={async () => {
              await refreshAllSubscriptions();
              message.success('全部订阅已拉取更新');
            }}
          >
            更新全部订阅
          </Button>
        }
      >
        <Card size="small" title="新增订阅分组" style={{ marginBottom: 16 }}>
          <Form
            form={subForm}
            layout="vertical"
            initialValues={{ autoUpdate: true, updateInterval: 12 }}
          >
            <Form.Item
              name="name"
              label="分组/订阅名称 (可选，留空将根据订阅域名自动命名)"
            >
              <Input placeholder="例如：机场A - 香港专线" />
            </Form.Item>
            <Form.Item
              name="url"
              label="订阅链接 URL (可选，留空则为纯本地分组)"
            >
              <Input placeholder="https://example.com/api/v1/client/subscribe?token=..." />
            </Form.Item>
            <Space size={24}>
              <Form.Item
                name="autoUpdate"
                label="定时自动更新"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item name="updateInterval" label="更新间隔 (小时)">
                <InputNumber min={1} max={168} />
              </Form.Item>
            </Space>
            <Button type="primary" block onClick={handleAddSub}>
              保存订阅分组
            </Button>
          </Form>
        </Card>

        <List
          header={<strong>当前订阅分组列表 ({subscriptions.length})</strong>}
          bordered
          dataSource={subscriptions}
          renderItem={(sub) => (
            <List.Item
              actions={[
                <Button
                  key="refresh"
                  size="small"
                  loading={refreshingSub}
                  icon={<ReloadOutlined />}
                  onClick={async () => {
                    try {
                      await refreshSubscription(sub.id);
                      message.success(`已刷新分组：${sub.name}`);
                    } catch (err: unknown) {
                      const msg =
                        err instanceof Error ? err.message : String(err);
                      message.error(`拉取失败: ${msg}`);
                    }
                  }}
                />,
                sub.id !== 'default' ? (
                  <Popconfirm
                    key="del"
                    title="确定删除该订阅分组及其下属所有节点？"
                    onConfirm={async () => {
                      await removeSubscription(sub.id);
                      message.success('已删除订阅分组');
                    }}
                  >
                    <Button size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                ) : null,
              ]}
            >
              <List.Item.Meta
                title={
                  <Space>
                    <span>{sub.name}</span>
                    <Tag color="blue">{sub.nodeCount} 节点</Tag>
                  </Space>
                }
                description={
                  <div style={{ fontSize: 12 }}>
                    <div>
                      {sub.url ? sub.url : '本地自定义节点分组'}
                    </div>
                    <div>上次更新：{formatTimestamp(sub.lastUpdated)}</div>
                    {sub.lastError && (
                      <div style={{ color: '#ff4d4f' }}>
                        最近错误：{sub.lastError}
                      </div>
                    )}
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Drawer>
    </Space>
  );
};
