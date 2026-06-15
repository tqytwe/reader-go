import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Switch, message } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
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

const SCOPES = ['all', 'title', 'content', 'toc', 'search']

export default function ReplaceRules() {
  const [rules, setRules] = useState<ReplaceRule[]>([])
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<ReplaceRule | null>(null)
  const [form] = Form.useForm()

  const load = async () => {
    const data: unknown = await api.getReplaceRules()
    setRules(Array.isArray(data) ? (data as ReplaceRule[]) : [])
  }

  useEffect(() => { load() }, [])

  const handleSave = async () => {
    const v = await form.validateFields()
    if (editing) {
      await api.updateReplaceRule(String(editing.id), v)
    } else {
      await api.createReplaceRule(v)
    }
    message.success('已保存')
    setOpen(false)
    load()
  }

  return (
    <div className="p-6">
      <div className="flex justify-between mb-4">
        <h1 className="text-2xl font-bold">替换规则</h1>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); form.resetFields(); setOpen(true) }}>
          新增
        </Button>
      </div>
      <Table
        rowKey="id"
        dataSource={rules}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '模式', dataIndex: 'pattern', ellipsis: true },
          { title: '范围', dataIndex: 'scope' },
          { title: '启用', dataIndex: 'enabled', render: (v: boolean) => (v ? '是' : '否') },
          {
            title: '操作',
            render: (_, r) => (
              <Button size="small" onClick={() => { setEditing(r); form.setFieldsValue(r); setOpen(true) }}>编辑</Button>
            ),
          },
        ]}
      />
      <Modal title={editing ? '编辑' : '新增'} open={open} onOk={handleSave} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical" initialValues={{ scope: 'content', enabled: true }}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="pattern" label="正则" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="replacement" label="替换为"><Input /></Form.Item>
          <Form.Item name="scope" label="范围"><Select options={SCOPES.map((s) => ({ label: s, value: s }))} /></Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
