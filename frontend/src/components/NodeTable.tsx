import React, { useEffect, useState } from 'react';
import {
  Button,
  Col,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleFilled,
  CopyOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  FolderOpenOutlined,
  SwapOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { config, ClipboardSetText } from '../api';
import { useConfigStore } from '../stores/configStore';
import { useNodeStore } from '../stores/nodeStore';
import { getDelayStatusColor, getProtocolTagColor } from '../utils/format';

interface NodeTableProps {
  dataSource: config.NodeConfig[];
  selectedRowKeys?: React.Key[];
  onSelectChange?: (keys: React.Key[]) => void;
}

interface ResizableHeaderCellProps
  extends React.HTMLAttributes<HTMLTableCellElement> {
  width?: number;
  minWidth?: number;
  onResizeWidth?: (nextWidth: number) => void;
}

const ResizableHeaderCell: React.FC<ResizableHeaderCellProps> = ({
  width,
  minWidth = 80,
  onResizeWidth,
  style,
  children,
  ...restProps
}) => {
  if (!width || !onResizeWidth) {
    return (
      <th style={style} {...restProps}>
        {children}
      </th>
    );
  }

  const handleMouseDown = (e: React.MouseEvent<HTMLSpanElement>) => {
    e.preventDefault();
    e.stopPropagation();
    const startX = e.clientX;
    const startWidth = width;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;
      const next = Math.max(minWidth, startWidth + deltaX);
      onResizeWidth(next);
    };

    const handleMouseUp = () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
    };

    document.body.style.cursor = 'col-resize';
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  return (
    <th
      {...restProps}
      style={{
        ...style,
        width,
        minWidth,
        position: 'relative',
        whiteSpace: 'nowrap',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
      }}
    >
      <div
        style={{
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          paddingRight: 8,
        }}
      >
        {children}
      </div>
      <span
        onMouseDown={handleMouseDown}
        onClick={(e) => e.stopPropagation()}
        title="左右拖动调整列宽"
        style={{
          position: 'absolute',
          right: 0,
          top: 0,
          bottom: 0,
          width: 8,
          cursor: 'col-resize',
          zIndex: 10,
          display: 'inline-block',
          borderRight: '2px solid rgba(148, 163, 184, 0.25)',
        }}
      />
    </th>
  );
};

const INITIAL_COL_WIDTHS: Record<string, { width: number; minWidth: number }> = {
  active: { width: 82, minWidth: 72 },
  name: { width: 200, minWidth: 120 },
  groupName: { width: 150, minWidth: 110 },
  protocol: { width: 320, minWidth: 140 },
  endpoint: { width: 185, minWidth: 120 },
  tcpDelay: { width: 135, minWidth: 105 },
  realDelay: { width: 145, minWidth: 110 },
  actions: { width: 175, minWidth: 155 },
};

export const NodeTable: React.FC<NodeTableProps> = ({
  dataSource,
  selectedRowKeys,
  onSelectChange,
}) => {
  const activeNodeId = useConfigStore((s) => s.appConfig.activeNodeId);
  const subscriptions = useNodeStore((s) => s.subscriptions);
  const selectActiveNode = useNodeStore((s) => s.selectActiveNode);
  const pingSingleNode = useNodeStore((s) => s.pingSingleNode);
  const updateNode = useNodeStore((s) => s.updateNode);
  const removeNode = useNodeStore((s) => s.removeNode);
  const moveNodesToGroup = useNodeStore((s) => s.moveNodesToGroup);

  const [colWidths, setColWidths] = useState<Record<string, number>>(() => {
    const map: Record<string, number> = {};
    Object.entries(INITIAL_COL_WIDTHS).forEach(([k, v]) => {
      map[k] = v.width;
    });
    return map;
  });

  const [editingNode, setEditingNode] = useState<config.NodeConfig | null>(null);
  const [editForm] = Form.useForm();
  const watchedProtocol = Form.useWatch('protocol', editForm) || 'vless';
  const watchedNetwork = Form.useWatch('network', editForm) || 'tcp';
  const watchedSecurity = Form.useWatch('security', editForm) || 'none';

  useEffect(() => {
    if (editingNode) {
      const sec =
        editingNode.security ||
        (editingNode.tls ? 'tls' : 'none');
      editForm.setFieldsValue({
        ...editingNode,
        protocol: (editingNode.protocol || 'vless').toLowerCase(),
        network: (editingNode.network || 'tcp').toLowerCase(),
        security: sec,
      });
    }
  }, [editingNode, editForm]);

  const openEditModal = (record: config.NodeConfig) => {
    setEditingNode(record);
  };

  const handleEditSave = async () => {
    if (!editingNode) return;
    try {
      const values = await editForm.validateFields();
      const sec = values.security || 'none';
      const updated = {
        ...editingNode,
        ...values,
        port: Number(values.port),
        alterId: Number(values.alterId || 0),
        upMbps: Number(values.upMbps || 0),
        downMbps: Number(values.downMbps || 0),
        security: sec,
        tls:
          sec === 'tls' ||
          sec === 'reality' ||
          values.protocol === 'trojan' ||
          values.protocol === 'hysteria2' ||
          values.protocol === 'tuic' ||
          Boolean(values.tls),
      } as config.NodeConfig;

      await updateNode(updated);
      message.success(`已保存节点配置：${updated.name}`);
      setEditingNode(null);
    } catch {
      // validation error
    }
  };

  const renderDelayTag = (delay: number, label: string) => {
    if (delay === 0) {
      return <Tag color="default">未测</Tag>;
    }
    if (delay < 0) {
      return <Tag color="error">超时</Tag>;
    }
    return (
      <Tooltip title={`${label}: ${delay} ms`}>
        <Tag color={getDelayStatusColor(delay)}>{delay} ms</Tag>
      </Tooltip>
    );
  };

  const buildGroupMenuItems = (record: config.NodeConfig) =>
    subscriptions.map((sub) => ({
      key: sub.id,
      icon: <FolderOpenOutlined />,
      disabled: sub.id === record.subId,
      label: (
        <span>
          {sub.name}{' '}
          {sub.id === record.subId ? (
            <span style={{ fontSize: 11, opacity: 0.6 }}>(当前分组)</span>
          ) : null}
        </span>
      ),
      onClick: async () => {
        await moveNodesToGroup([record.id], sub.id);
        message.success(`已将节点「${record.name}」移动至分组「${sub.name}」`);
      },
    }));

  const cellEllipsisStyle: React.CSSProperties = {
    width: '100%',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  };

  const baseColumns: ColumnsType<config.NodeConfig> = [
    {
      title: '状态',
      key: 'active',
      align: 'center',
      ellipsis: true,
      render: (_, record) =>
        record.id === activeNodeId ? (
          <Tag color="success" icon={<CheckCircleFilled />} style={{ margin: 0 }}>
            当前
          </Tag>
        ) : (
          <Button
            type="link"
            size="small"
            onClick={async () => {
              await selectActiveNode(record.id);
              message.success(`已切换活动节点为：${record.name}`);
            }}
          >
            选用
          </Button>
        ),
    },
    {
      title: '节点名称 (点击可编辑)',
      dataIndex: 'name',
      key: 'name',
      ellipsis: { showTitle: false },
      render: (name: string, record) => (
        <Tooltip title={`${name} (点击编辑节点配置)`}>
          <div
            style={{
              ...cellEllipsisStyle,
              cursor: 'pointer',
              fontWeight: record.id === activeNodeId ? 600 : 400,
            }}
            onClick={() => openEditModal(record)}
          >
            {name}
          </div>
        </Tooltip>
      ),
    },
    {
      title: '所属分组 (点击可调整)',
      dataIndex: 'groupName',
      key: 'groupName',
      ellipsis: { showTitle: false },
      render: (groupName: string, record) => (
        <div style={cellEllipsisStyle}>
          <Dropdown
            trigger={['click']}
            menu={{ items: buildGroupMenuItems(record) }}
          >
            <Tooltip title="点击切换/移动该节点所属分组">
              <Tag
                color="blue"
                style={{
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  margin: 0,
                  maxWidth: '100%',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                <span
                  style={{
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {groupName || '默认分组'}
                </span>
                <DownOutlined style={{ fontSize: 10, flexShrink: 0 }} />
              </Tag>
            </Tooltip>
          </Dropdown>
        </div>
      ),
    },
    {
      title: '协议 / 传输 (点击可编辑)',
      key: 'protocol',
      ellipsis: { showTitle: false },
      render: (_, record) => {
        const tags: React.ReactNode[] = [
          <Tag
            key="proto"
            color={getProtocolTagColor(record.protocol)}
            style={{ margin: 0, textAlign: 'center' }}
          >
            {record.protocol.toUpperCase()}
          </Tag>,
        ];
        if (record.network) {
          tags.push(
            <Tag key="net" style={{ margin: 0, textAlign: 'center' }}>
              {record.network.toUpperCase()}
            </Tag>
          );
        }
        if (record.security === 'reality') {
          tags.push(
            <Tag
              key="sec"
              color="gold"
              style={{ margin: 0, textAlign: 'center' }}
            >
              REALITY
            </Tag>
          );
        } else if (record.tls) {
          tags.push(
            <Tag
              key="tls"
              color="green"
              style={{ margin: 0, textAlign: 'center' }}
            >
              TLS
            </Tag>
          );
        }
        if (record.flow) {
          tags.push(
            <Tag
              key="flow"
              color="cyan"
              style={{ margin: 0, textAlign: 'center' }}
            >
              {record.flow.includes('vision')
                ? 'VISION'
                : record.flow.toUpperCase()}
            </Tag>
          );
        }
        if (record.mldsa65Verify) {
          tags.push(
            <Tag
              key="pqv"
              color="purple"
              style={{ margin: 0, textAlign: 'center' }}
            >
              PQV
            </Tag>
          );
        }
        if (record.fingerprint) {
          tags.push(
            <Tag
              key="fp"
              color="geekblue"
              style={{ margin: 0, textAlign: 'center' }}
            >
              {record.fingerprint.toUpperCase()}
            </Tag>
          );
        }

        return (
          <Tooltip title="点击编辑协议与传输层参数">
            <div
              onClick={() => openEditModal(record)}
              style={{
                width: '100%',
                overflow: 'hidden',
                whiteSpace: 'nowrap',
                cursor: 'pointer',
              }}
            >
              <div
                style={{
                  display: 'inline-flex',
                  flexWrap: 'nowrap',
                  gap: 6,
                  alignItems: 'center',
                  maxWidth: '100%',
                }}
              >
                {tags}
              </div>
            </div>
          </Tooltip>
        );
      },
    },
    {
      title: '服务器地址 (点击可编辑)',
      key: 'endpoint',
      ellipsis: { showTitle: false },
      render: (_, record) => (
        <Tooltip title={`${record.server}:${record.port} (点击编辑连接地址与端口)`}>
          <div
            onClick={() => openEditModal(record)}
            style={{
              ...cellEllipsisStyle,
              fontFamily: 'monospace',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            {record.server}:{record.port}
          </div>
        </Tooltip>
      ),
    },
    {
      title: 'TCP 握手延迟',
      dataIndex: 'tcpDelay',
      key: 'tcpDelay',
      ellipsis: true,
      sorter: (a, b) => {
        const da = a.tcpDelay > 0 ? a.tcpDelay : 999999;
        const db = b.tcpDelay > 0 ? b.tcpDelay : 999999;
        return da - db;
      },
      render: (tcpDelay: number) => (
        <div style={cellEllipsisStyle}>
          {renderDelayTag(tcpDelay, 'TCP 物理握手延迟')}
        </div>
      ),
    },
    {
      title: 'HTTP 204 延迟',
      dataIndex: 'realDelay',
      key: 'realDelay',
      ellipsis: true,
      sorter: (a, b) => {
        const da = a.realDelay > 0 ? a.realDelay : 999999;
        const db = b.realDelay > 0 ? b.realDelay : 999999;
        return da - db;
      },
      render: (realDelay: number) => (
        <div style={cellEllipsisStyle}>
          {renderDelayTag(realDelay, 'HTTP 204 真实连通延迟')}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      align: 'right',
      ellipsis: true,
      render: (_, record) => (
        <Space size={4} style={{ flexWrap: 'nowrap' }}>
          <Tooltip title="编辑节点完整参数">
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEditModal(record)}
            />
          </Tooltip>
          <Tooltip title="测试该节点真实延迟">
            <Button
              size="small"
              icon={<ThunderboltOutlined />}
              onClick={async () => {
                message.loading({
                  content: `正在测量 ${record.name} 延迟...`,
                  key: `ping-${record.id}`,
                });
                const ms = await pingSingleNode(record.id);
                if (ms > 0) {
                  message.success({
                    content: `${record.name}: TCP 握手 ${ms} ms`,
                    key: `ping-${record.id}`,
                  });
                } else {
                  message.warning({
                    content: `${record.name}: 连接超时`,
                    key: `ping-${record.id}`,
                  });
                }
              }}
            />
          </Tooltip>
          <Dropdown
            trigger={['click']}
            menu={{ items: buildGroupMenuItems(record) }}
          >
            <Tooltip title="移动至其他分组">
              <Button size="small" icon={<SwapOutlined />} />
            </Tooltip>
          </Dropdown>
          {record.rawLink && (
            <Tooltip title="复制节点分享链接">
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={async () => {
                  await ClipboardSetText(record.rawLink || '');
                  message.success('已复制节点分享链接');
                }}
              />
            </Tooltip>
          )}
          <Popconfirm
            title="确定删除该节点？"
            onConfirm={async () => {
              await removeNode(record.id);
              message.success('节点已删除');
            }}
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const columns: ColumnsType<config.NodeConfig> = baseColumns.map((col) => {
    const colKey = String(col.key || '');
    const cfg = INITIAL_COL_WIDTHS[colKey] || { width: 140, minWidth: 90 };
    const currentWidth = colWidths[colKey] || cfg.width;

    return {
      ...col,
      width: currentWidth,
      onHeaderCell: () =>
        ({
          width: currentWidth,
          minWidth: cfg.minWidth,
          onResizeWidth: (nextW: number) => {
            setColWidths((prev) => ({
              ...prev,
              [colKey]: Math.max(cfg.minWidth, nextW),
            }));
          },
        } as React.HTMLAttributes<HTMLElement>),
      onCell: () => ({
        style: {
          maxWidth: currentWidth,
          minWidth: cfg.minWidth,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        },
      }),
    };
  });

  return (
    <>
      <Table<config.NodeConfig>
        rowKey="id"
        size="middle"
        tableLayout="fixed"
        style={{ width: '100%' }}
        components={{
          header: {
            cell: ResizableHeaderCell,
          },
        }}
        columns={columns}
        dataSource={dataSource}
        scroll={{ x: 'max-content' }}
        rowSelection={
          onSelectChange
            ? {
                selectedRowKeys,
                onChange: onSelectChange,
              }
            : undefined
        }
        pagination={{ pageSize: 10, showSizeChanger: false }}
      />

      {/* 通用多协议节点编辑器弹窗 */}
      <Modal
        open={Boolean(editingNode)}
        title={`编辑节点配置 — ${editingNode?.name || ''}`}
        width={780}
        centered
        okText="保存并应用"
        cancelText="取消"
        onOk={handleEditSave}
        onCancel={() => setEditingNode(null)}
        styles={{
          body: {
            maxHeight: '72vh',
            overflowY: 'auto',
            paddingRight: 4,
          },
        }}
      >
        <Form
          form={editForm}
          className="node-edit-form"
          layout="vertical"
          size="middle"
        >
          <div className="node-edit-section">
            <div className="node-edit-section-title">
              1. 基础连接与分组信息 (通用)
            </div>
            <Row gutter={16}>
              <Col span={10}>
                <Form.Item
                  name="name"
                  label="节点名称 / 备注"
                  rules={[{ required: true, message: '请输入节点名称' }]}
                >
                  <Input placeholder="例如：香港 IEPL 01" />
                </Form.Item>
              </Col>
              <Col span={7}>
                <Form.Item name="subId" label="所属订阅分组">
                  <Select
                    options={subscriptions.map((s) => ({
                      label: s.name,
                      value: s.id,
                    }))}
                  />
                </Form.Item>
              </Col>
              <Col span={7}>
                <Form.Item
                  name="protocol"
                  label="代理协议 (Protocol)"
                  rules={[{ required: true }]}
                >
                  <Select
                    options={[
                      { label: 'VLESS', value: 'vless' },
                      { label: 'VMess', value: 'vmess' },
                      { label: 'Trojan', value: 'trojan' },
                      { label: 'Shadowsocks (SS)', value: 'ss' },
                      { label: 'Hysteria2 (HY2)', value: 'hysteria2' },
                      { label: 'TUIC v5', value: 'tuic' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={16}>
              <Col span={16}>
                <Form.Item
                  name="server"
                  label="服务器地址 (域名或 IP)"
                  rules={[{ required: true, message: '请输入服务器地址' }]}
                >
                  <Input placeholder="example.com 或 1.2.3.4" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  name="port"
                  label="端口 (Port)"
                  rules={[{ required: true, message: '请输入端口' }]}
                >
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </div>

          <div className="node-edit-section">
            <div className="node-edit-section-title">
              2. 协议认证与专属参数 ({watchedProtocol.toUpperCase()})
            </div>

            {/* VLESS 认证参数 */}
            {watchedProtocol === 'vless' && (
              <Row gutter={16}>
                <Col span={15}>
                  <Form.Item
                    name="uuid"
                    label="UUID (用户标识)"
                    rules={[{ required: true, message: '请输入 VLESS UUID' }]}
                  >
                    <Input placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
                  </Form.Item>
                </Col>
                <Col span={9}>
                  <Form.Item name="flow" label="XTLS 流控 (Flow)">
                    <Select
                      allowClear
                      placeholder="无流控 (默认)"
                      options={[
                        { label: '无 (none)', value: '' },
                        {
                          label: 'xtls-rprx-vision (VISION 推荐)',
                          value: 'xtls-rprx-vision',
                        },
                      ]}
                    />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {/* VMess 认证参数 */}
            {watchedProtocol === 'vmess' && (
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="uuid"
                    label="UUID"
                    rules={[{ required: true, message: '请输入 VMess UUID' }]}
                  >
                    <Input placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item name="cipher" label="加密算法 (Security)">
                    <Select
                      options={[
                        { label: 'auto (推荐)', value: 'auto' },
                        { label: 'aes-128-gcm', value: 'aes-128-gcm' },
                        {
                          label: 'chacha20-poly1305',
                          value: 'chacha20-poly1305',
                        },
                        { label: 'none', value: 'none' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item name="alterId" label="AlterID">
                    <InputNumber min={0} max={64} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {/* Trojan 认证参数 */}
            {watchedProtocol === 'trojan' && (
              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item
                    name="password"
                    label="Trojan 密码 (Password)"
                    rules={[{ required: true, message: '请输入密码' }]}
                  >
                    <Input.Password placeholder="输入 Trojan 连接密码" />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {/* Shadowsocks 认证参数 */}
            {watchedProtocol === 'ss' && (
              <Row gutter={16}>
                <Col span={10}>
                  <Form.Item name="cipher" label="SS 加密方式 (Method)">
                    <Select
                      options={[
                        { label: 'aes-256-gcm', value: 'aes-256-gcm' },
                        { label: 'aes-128-gcm', value: 'aes-128-gcm' },
                        {
                          label: 'chacha20-ietf-poly1305',
                          value: 'chacha20-ietf-poly1305',
                        },
                        {
                          label: '2022-blake3-aes-256-gcm',
                          value: '2022-blake3-aes-256-gcm',
                        },
                        {
                          label: '2022-blake3-aes-128-gcm',
                          value: '2022-blake3-aes-128-gcm',
                        },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col span={14}>
                  <Form.Item
                    name="password"
                    label="SS 密码 (Password)"
                    rules={[{ required: true, message: '请输入 SS 密码' }]}
                  >
                    <Input.Password placeholder="输入 Shadowsocks 密码" />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {/* Hysteria2 认证与带宽参数 */}
            {watchedProtocol === 'hysteria2' && (
              <>
                <Row gutter={16}>
                  <Col span={10}>
                    <Form.Item
                      name="password"
                      label="认证密码 (Auth Password)"
                      rules={[
                        { required: true, message: '请输入 Hysteria2 密码' },
                      ]}
                    >
                      <Input.Password placeholder="Hysteria2 密码" />
                    </Form.Item>
                  </Col>
                  <Col span={7}>
                    <Form.Item name="obfsType" label="混淆类型 (Obfs)">
                      <Select
                        options={[
                          { label: '无混淆', value: '' },
                          { label: 'salamander', value: 'salamander' },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={7}>
                    <Form.Item name="obfsPassword" label="混淆密码">
                      <Input placeholder="可选 salamander 密码" />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name="upMbps" label="上行带宽限制 (Mbps)">
                      <InputNumber
                        min={0}
                        max={10000}
                        style={{ width: '100%' }}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item name="downMbps" label="下行带宽限制 (Mbps)">
                      <InputNumber
                        min={0}
                        max={10000}
                        style={{ width: '100%' }}
                      />
                    </Form.Item>
                  </Col>
                </Row>
              </>
            )}

            {/* TUIC 认证参数 */}
            {watchedProtocol === 'tuic' && (
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="uuid"
                    label="TUIC UUID"
                    rules={[{ required: true, message: '请输入 TUIC UUID' }]}
                  >
                    <Input placeholder="UUID" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="password"
                    label="TUIC 密码"
                    rules={[{ required: true, message: '请输入 TUIC 密码' }]}
                  >
                    <Input.Password placeholder="Password" />
                  </Form.Item>
                </Col>
              </Row>
            )}
          </div>

          <div className="node-edit-section" style={{ marginBottom: 4 }}>
            <div className="node-edit-section-title">
              3. 传输层与 TLS / Reality 安全伪装
            </div>

            <Row gutter={16}>
              <Col span={8}>
                <Form.Item name="network" label="传输协议 (Network)">
                  <Select
                    options={[
                      { label: 'TCP (默认)', value: 'tcp' },
                      { label: 'WebSocket (WS)', value: 'ws' },
                      { label: 'gRPC', value: 'grpc' },
                      { label: 'QUIC', value: 'quic' },
                      { label: 'HTTP/2 (h2)', value: 'h2' },
                      { label: 'HTTPUpgrade', value: 'httpupgrade' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="security" label="安全层类型 (Security)">
                  <Select
                    options={[
                      { label: '无加密 (none)', value: 'none' },
                      { label: 'TLS', value: 'tls' },
                      { label: 'REALITY (Xray/Sing-box)', value: 'reality' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="fingerprint" label="uTLS 浏览器指纹 (fp)">
                  <Select
                    allowClear
                    placeholder="默认 (chrome)"
                    options={[
                      { label: 'chrome (推荐)', value: 'chrome' },
                      { label: 'firefox', value: 'firefox' },
                      { label: 'safari', value: 'safari' },
                      { label: 'ios', value: 'ios' },
                      { label: 'edge', value: 'edge' },
                      { label: 'random', value: 'random' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            {(watchedNetwork === 'ws' ||
              watchedNetwork === 'h2' ||
              watchedNetwork === 'httpupgrade') && (
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="path" label="请求路径 (Path)">
                    <Input placeholder="/ws" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="host" label="伪装域名头 (Host)">
                    <Input placeholder="cdn.example.com" />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {watchedNetwork === 'grpc' && (
              <Row gutter={16}>
                <Col span={24}>
                  <Form.Item name="serviceName" label="gRPC ServiceName">
                    <Input placeholder="grpc-service" />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {(watchedSecurity === 'tls' || watchedSecurity === 'reality') && (
              <Row gutter={16}>
                <Col span={14}>
                  <Form.Item name="sni" label="TLS SNI (Server Name)">
                    <Input placeholder="www.microsoft.com 或 www.cloudflare.com" />
                  </Form.Item>
                </Col>
                <Col span={10}>
                  <Form.Item name="alpn" label="ALPN (逗号分隔)">
                    <Input placeholder="h2,http/1.1" />
                  </Form.Item>
                </Col>
              </Row>
            )}

            {watchedSecurity === 'reality' && (
              <>
                <Row gutter={16}>
                  <Col span={14}>
                    <Form.Item name="publicKey" label="Reality 公钥 (pbk)">
                      <Input placeholder="Reality PublicKey" />
                    </Form.Item>
                  </Col>
                  <Col span={5}>
                    <Form.Item name="shortId" label="ShortId (sid)">
                      <Input placeholder="c868" />
                    </Form.Item>
                  </Col>
                  <Col span={5}>
                    <Form.Item name="spiderX" label="SpiderX (spx)">
                      <Input placeholder="/path" />
                    </Form.Item>
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={24}>
                    <Form.Item
                      name="mldsa65Verify"
                      label="后量子验证公钥 (pqv / ML-DSA-65，可选)"
                    >
                      <Input placeholder="Xray Reality 后量子签名验证公钥 (pqv)" />
                    </Form.Item>
                  </Col>
                </Row>
              </>
            )}
          </div>
        </Form>
      </Modal>
    </>
  );
};
