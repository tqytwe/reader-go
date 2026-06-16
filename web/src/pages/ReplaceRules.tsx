import { useEffect, useState, useMemo } from 'react'
import {
  Table, Button, Modal, Form, Input, Select, Switch, message,
  Space, Popconfirm, Tag, Empty, Upload, Tooltip,
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined,
  UploadOutlined, DownloadOutlined, CheckCircleOutlined, CloseCircleOutlined,
} from '@ant-design/icons'
import { api } from '../api/client'

interface ReplaceRule {
  id: number
  name: string
  pattern: string
  replacement: string
  scope: string
  caseInsensitive: boolean
  enabled: boolean
}

const SCOPES = [
  { label: '全部', value: 'all' },
  { label: '标题', value: 'title' },
  { label: '内容', value: 'content' },
  { label: '目录', value: 'toc' },
  { label: '搜索', value: 'search' },
]

const SCOPE_COLORS: Record<string, string> = {
  all: 'default',
  title: 'blue',
  content: 'green',
  toc: 'orange',
  search: 'purple',
}

export default function ReplaceRules() {
  const [rules, setRules] = useState<ReplaceRule[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<ReplaceRule | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const [searchQuery, setSearchQuery] = useState('')
  const [scopeFilter, setScopeFilter] = useState<string>('')
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])

  const load = async () => {
    setLoading(true)
    try {
      const data: unknown = await api.getReplaceRules()
      setRules(Array.isArray(data) ? (data as ReplaceRule[]) : [])
    } catch {
      message.error('加载替换规则失败')
      setRules([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  // 过滤后的规则
  const filteredRules = useMemo(() => {
    let result = rules
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(r =>
        r.name.toLowerCase().includes(q) ||
        r.pattern.toLowerCase().includes(q)
      )
    }
    if (scopeFilter) {
      result = result.filter(r => r.scope === scopeFilter)
    }
    return result
  }, [rules, searchQuery, scopeFilter])

  const handleSave = async () => {
    try {
      const v = await form.validateFields()
      setSaving(true)
      if (editing) {
        await api.updateReplaceRule(String(editing.id), v)
        message.success('规则已更新')
      } else {
        await api.createReplaceRule(v)
        message.success('规则已创建')
      }
      setOpen(false)
      form.resetFields()
      setEditing(null)
      await load()
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'errorFields' in err) return // form validation
      message.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await api.deleteReplaceRule(String(id))
      message.success('已删除')
      await load()
    } catch {
      message.error('删除失败')
    }
  }

  const handleToggleEnabled = async (rule: ReplaceRule) => {
    try {
      await api.updateReplaceRule(String(rule.id), { ...rule, enabled: !rule.enabled })
      message.success(rule.enabled ? '已禁用' : '已启用')
      await load()
    } catch {
      message.error('操作失败')
    }
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的规则')
      return
    }
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 条规则吗？`,
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await Promise.all(selectedRowKeys.map(id => api.deleteReplaceRule(String(id))))
          setSelectedRowKeys([])
          message.success(`已删除 ${selectedRowKeys.length} 条规则`)
          await load()
        } catch {
          message.error('批量删除失败')
        }
      },
    })
  }

  const handleExport = () => {
    const dataStr = JSON.stringify(rules, null, 2)
    const blob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `replace-rules-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  }

  const handleImport = (file: File) => {
    const reader = new FileReader()
    reader.onload = async () => {
      try {
        const data = JSON.parse(reader.result as string)
        const list = Array.isArray(data) ? data : [data]
        let imported = 0
        for (const rule of list) {
          try {
            await api.createReplaceRule(rule)
            imported++
          } catch {
            // skip
          }
        }
        message.success(`导入 ${imported} 条规则`)
        await load()
      } catch {
        message.error('导入失败，请检查 JSON 格式')
      }
    }
    reader.readAsText(file)
    return false
  }

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      ellipsis: true,
    },
    {
      title: '正则',
      dataIndex: 'pattern',
      key: 'pattern',
      ellipsis: true,
      render: (pattern: string) => (
        <Tooltip title={pattern}>
          <code className="text-xs bg-gray-100 dark:bg-gray-700 px-1 rounded">{pattern}</code>
        </Tooltip>
      ),
    },
    {
      title: '替换为',
      dataIndex: 'replacement',
      key: 'replacement',
      width: 150,
      ellipsis: true,
      render: (v: string) => v || <span className="text-gray-400">（空）</span>,
    },
    {
      title: '范围',
      dataIndex: 'scope',
      key: 'scope',
      width: 80,
      render: (scope: string) => (
        <Tag color={SCOPE_COLORS[scope] || 'default'}>
          {SCOPES.find(s => s.value === scope)?.label || scope}
        </Tag>
      ),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean, record: ReplaceRule) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggleEnabled(record)}
          checkedChildren={<CheckCircleOutlined />}
          unCheckedChildren={<CloseCircleOutlined />}
          size="small"
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, record: ReplaceRule) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => { setEditing(record); form.setFieldsValue(record); setOpen(true) }}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这条规则吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">替换规则</h1>
        <Space>
          <Button
            danger
            icon={<DeleteOutlined />}
            onClick={handleBatchDelete}
            disabled={selectedRowKeys.length === 0}
          >
            批量删除 {selectedRowKeys.length > 0 && `(${selectedRowKeys.length})`}
          </Button>
          <Upload accept=".json" showUploadList={false} beforeUpload={handleImport}>
            <Button icon={<UploadOutlined />}>导入</Button>
          </Upload>
          <Button
            icon={<DownloadOutlined />}
            onClick={handleExport}
            disabled={rules.length === 0}
          >
            导出
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => { setEditing(null); form.resetFields(); setOpen(true) }}
          >
            新增
          </Button>
        </Space>
      </div>

      {/* 搜索和筛选 */}
      <div className="flex gap-4 mb-4">
        <Input
          placeholder="搜索名称、正则..."
          prefix={<SearchOutlined />}
          allowClear
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{ width: 280 }}
        />
        <Select
          placeholder="筛选范围"
          allowClear
          value={scopeFilter || undefined}
          onChange={(v) => setScopeFilter(v || '')}
          style={{ width: 140 }}
          options={SCOPES}
        />
        <span className="ml-auto text-gray-500 self-center text-sm">
          共 {rules.length} 条规则{filteredRules.length !== rules.length && `，显示 ${filteredRules.length} 条`}
        </span>
      </div>

      {/* 表格 */}
      {rules.length === 0 && !loading ? (
        <div className="text-center py-16">
          <Empty
            description="暂无替换规则"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => { setEditing(null); form.resetFields(); setOpen(true) }}
            >
              创建第一条规则
            </Button>
          </Empty>
        </div>
      ) : (
        <Table
          rowKey="id"
          dataSource={filteredRules}
          columns={columns}
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys),
          }}
          pagination={{
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          size="middle"
        />
      )}

      {/* 编辑/新增弹窗 */}
      <Modal
        title={editing ? '编辑规则' : '新增规则'}
        open={open}
        onOk={handleSave}
        onCancel={() => { setOpen(false); setEditing(null); form.resetFields() }}
        confirmLoading={saving}
        okText={editing ? '保存' : '创建'}
        cancelText="取消"
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ scope: 'content', enabled: true, caseInsensitive: true }}
          className="mt-4"
        >
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例如：去除广告" />
          </Form.Item>
          <Form.Item
            name="pattern"
            label="正则表达式"
            rules={[{ required: true, message: '请输入正则表达式' }]}
            extra="使用正则表达式匹配需要替换的文本"
          >
            <Input placeholder="例如：<!--.*?-->" />
          </Form.Item>
          <Form.Item name="replacement" label="替换为" extra="匹配到的文本将被替换为此内容，留空则表示删除">
            <Input placeholder="留空表示删除匹配内容" />
          </Form.Item>
          <div className="flex gap-4">
            <Form.Item name="scope" label="作用范围" className="flex-1">
              <Select options={SCOPES} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="caseInsensitive" label="忽略大小写" valuePropName="checked">
              <Switch />
            </Form.Item>
          </div>
        </Form>
      </Modal>
    </div>
  )
}
