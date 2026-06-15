import {
  Table,
  Modal,
  Button,
  Upload,
  message,
  Space,
  Popconfirm,
  Tag,
  Input,
  Select,
  Switch,
  Checkbox,
  Popover,
  Dropdown,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UploadOutlined,
  DownloadOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LinkOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { useState, useMemo, useEffect } from 'react'
import { Form } from 'antd'
import { api } from '@/api/client'
import { useStore, type BookSource } from '@/store/useStore'
import BookSourceFormModal from '@/components/BookSourceFormModal'
import {
  EMPTY_BOOK_SOURCE,
  normalizeBookSource,
  parseBookSourceHeaders,
  type BookSourceDTO,
  type SourceStat,
  type ParseMode,
} from '@/types/booksource'

const { Search } = Input
const { Option } = Select

function unwrapBookSourceCollection(data: unknown): unknown[] {
  if (Array.isArray(data)) {
    return data
  }
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>
    if (Array.isArray(obj.bookSources)) {
      return obj.bookSources
    }
    if (Array.isArray(obj.sources)) {
      return obj.sources
    }
    return [data]
  }
  return []
}

export default function BookSourceManage() {
  const {
    bookSources,
    bookSourcesLoading,
    bookSourceSearchKeyword,
    bookSourceGroupFilter,
    setBookSources,
    setBookSourcesLoading,
    setBookSourceSearchKeyword,
    setBookSourceGroupFilter,
    addBookSource,
    updateBookSource,
    removeBookSource,
    toggleBookSourceEnabled,
  } = useStore()

  const [form] = Form.useForm<BookSourceDTO>()
  const [modalVisible, setModalVisible] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [confirmLoading, setConfirmLoading] = useState(false)
  const [importLoading, setImportLoading] = useState(false)
  const [collectionModalVisible, setCollectionModalVisible] = useState(false)
  const [collectionUrl, setCollectionUrl] = useState('')
  const [collectionLoading, setCollectionLoading] = useState(false)
  const [enableOnlyNonJS, setEnableOnlyNonJS] = useState(true)
  const [collectionSettingsOpen, setCollectionSettingsOpen] = useState(false)
  const [batchLoading, setBatchLoading] = useState(false)
  const [sourceStats, setSourceStats] = useState<Record<number, SourceStat>>({})
  const [rawSources, setRawSources] = useState<BookSourceDTO[]>([])

  // 加载书源列表
  const fetchBookSources = async () => {
    setBookSourcesLoading(true)
    try {
      const data = await api.getBookSources()
      const list = Array.isArray(data) ? data : []
      const normalized = list.map((item) => normalizeBookSource(item as Record<string, unknown>))
      setRawSources(normalized)
      setBookSources(
        normalized.map((s) => ({
          id: String(s.id),
          name: s.name,
          baseUrl: s.baseUrl,
          searchUrl: s.searchUrl,
          searchRule: s.searchRule,
          mode: s.searchMode ?? 'default',
          enabled: s.enabled,
          group: s.group ?? '',
          headers: parseBookSourceHeaders(s.headers),
        }))
      )
      try {
        const stats = (await api.getBookSourceStats()) as unknown as SourceStat[]
        const map: Record<number, SourceStat> = {}
        stats.forEach((st) => { map[st.sourceId] = st })
        setSourceStats(map)
      } catch {
        setSourceStats({})
      }
    } catch (err) {
      console.error('Failed to fetch book sources:', err)
      message.error('加载书源列表失败')
      setBookSources([])
      setRawSources([])
    } finally {
      setBookSourcesLoading(false)
    }
  }

  useEffect(() => {
    fetchBookSources()
  }, [])

  // 筛选后的书源列表
  const filteredSources = useMemo(() => {
    let result = bookSources

    if (bookSourceSearchKeyword) {
      const kw = bookSourceSearchKeyword.toLowerCase()
      result = result.filter(
        (s: BookSource) =>
          s.name.toLowerCase().includes(kw) ||
          s.baseUrl.toLowerCase().includes(kw) ||
          s.group.toLowerCase().includes(kw)
      )
    }

    if (bookSourceGroupFilter) {
      result = result.filter((s: BookSource) => s.group === bookSourceGroupFilter)
    }

    return result
  }, [bookSources, bookSourceSearchKeyword, bookSourceGroupFilter])

  // 获取所有分组
  const groups = useMemo(() => {
    const set = new Set(bookSources.map((s: BookSource) => s.group).filter(Boolean))
    return Array.from(set).sort()
  }, [bookSources])

  // 打开添加模态框
  const handleAdd = () => {
    setEditingId(null)
    form.setFieldsValue({ ...EMPTY_BOOK_SOURCE })
    setModalVisible(true)
  }

  const handleEdit = (record: { id: string }) => {
    const full = rawSources.find((s) => String(s.id) === record.id)
    if (!full) {
      message.error('未找到完整书源数据，请刷新后重试')
      return
    }
    setEditingId(record.id)
    form.setFieldsValue(full)
    setModalVisible(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setConfirmLoading(true)

      if (values.headers) {
        try {
          JSON.parse(values.headers)
        } catch {
          message.error('请求头 JSON 格式错误')
          setConfirmLoading(false)
          return
        }
      }

      const payload: BookSourceDTO = { ...values, headers: values.headers || '{}' }

      if (editingId) {
        await api.updateBookSource(editingId, payload)
        updateBookSource(editingId, {
          name: payload.name,
          baseUrl: payload.baseUrl,
          searchUrl: payload.searchUrl,
          searchRule: payload.searchRule,
          mode: payload.searchMode ?? 'default',
          enabled: payload.enabled,
          group: payload.group ?? '',
          headers: JSON.parse(payload.headers || '{}'),
        })
        message.success('书源更新成功')
      } else {
        const res = (await api.createBookSource(payload)) as unknown as BookSourceDTO
        addBookSource({
          id: String(res?.id ?? Date.now()),
          name: payload.name,
          baseUrl: payload.baseUrl,
          searchUrl: payload.searchUrl,
          searchRule: payload.searchRule,
          mode: payload.searchMode ?? 'default',
          enabled: payload.enabled,
          group: payload.group ?? '',
          headers: JSON.parse(payload.headers || '{}'),
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        })
        message.success('书源添加成功')
      }

      setModalVisible(false)
      form.resetFields()
      await fetchBookSources()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存失败，请检查输入')
    } finally {
      setConfirmLoading(false)
    }
  }

  // 删除书源
  const handleDelete = async (id: string) => {
    try {
      await api.deleteBookSource(id)
      removeBookSource(id)
      message.success('删除成功')
    } catch {
      message.error('删除失败')
    }
  }

  // 切换启用状态
  const handleToggleEnabled = async (id: string, current: boolean) => {
    const full = rawSources.find((s) => String(s.id) === id)
    if (!full) {
      message.error('书源数据未加载完整')
      return
    }
    try {
      const payload = { ...full, enabled: !current }
      await api.updateBookSource(id, payload)
      toggleBookSourceEnabled(id)
      message.success(!current ? '已启用' : '已禁用')
      await fetchBookSources()
    } catch {
      message.error('操作失败')
    }
  }

  // 导入书源
  const handleImport = async (file: File) => {
    setImportLoading(true)
    const reader = new FileReader()
    reader.onload = async () => {
      try {
        const data = JSON.parse(reader.result as string)
        const sources = unwrapBookSourceCollection(data)
        if (sources.length === 0) {
          message.error('未找到可导入的书源')
          return
        }
        const result = await api.importBookSources(sources) as { imported?: number; failed?: number; errors?: string[] }
        fetchBookSources()
        if (result?.errors?.length) {
          message.warning(`导入 ${result.imported ?? 0} 个，失败 ${result.failed ?? 0} 个`)
          console.warn('import errors:', result.errors)
        } else {
          message.success(`成功导入 ${result?.imported ?? sources.length} 个书源`)
        }
      } catch {
        message.error('导入失败，请检查 JSON 格式')
      } finally {
        setImportLoading(false)
      }
    }
    reader.readAsText(file)
    return false // 阻止默认上传行为
  }

  // 链接导入书源合集
  const handleCollectionImport = async () => {
    if (!collectionUrl.trim()) {
      message.warning('请输入书源合集链接')
      return
    }
    setCollectionLoading(true)
    try {
      const result = await api.importBookSourceCollection(collectionUrl.trim(), enableOnlyNonJS) as {
        imported?: number
        total?: number
        failed?: number
        enabled?: number
        disabled?: number
        jsRequired?: number
        nonJs?: number
        errors?: string[]
      }
      await fetchBookSources()
      const summary = enableOnlyNonJS
        ? `导入 ${result?.imported ?? 0} 个，已启用 ${result?.enabled ?? 0} 个（无 JS ${result?.nonJs ?? 0} / 需 JS ${result?.jsRequired ?? 0}）`
        : `成功导入 ${result?.imported ?? 0} / ${result?.total ?? 0} 个书源`
      if (result?.errors?.length) {
        message.warning(`${summary}，${result.failed ?? 0} 失败`)
        console.warn('collection import errors:', result.errors)
      } else {
        message.success(summary)
      }
      setCollectionModalVisible(false)
      setCollectionUrl('')
    } catch (err: any) {
      message.error(err?.message || '链接导入失败')
    } finally {
      setCollectionLoading(false)
    }
  }

  const handleBatchEnable = async (
    payload: {
      target?: 'js' | 'nonJs' | 'all'
      enabled?: boolean
      enableOnlyNonJS?: boolean
    },
    label: string
  ) => {
    setBatchLoading(true)
    try {
      const result = await api.batchSetBookSourceEnabled(payload) as {
        updated?: number
        enabled?: number
        disabled?: number
        jsRequired?: number
        nonJs?: number
      }
      await fetchBookSources()
      message.success(
        `${label}：更新 ${result?.updated ?? 0} 个（已启用 ${result?.enabled ?? 0} / 已禁用 ${result?.disabled ?? 0}，无 JS ${result?.nonJs ?? 0} / 需 JS ${result?.jsRequired ?? 0}）`
      )
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '批量操作失败')
    } finally {
      setBatchLoading(false)
    }
  }

  const batchMenuItems: MenuProps['items'] = [
    {
      key: 'enableNonJsOnly',
      label: '仅启用无 JS 源（推荐）',
      onClick: () => handleBatchEnable({ enableOnlyNonJS: true }, '已应用仅启用无 JS 策略'),
    },
    { type: 'divider' },
    {
      key: 'enableNonJs',
      label: '批量启用无 JS 源',
      onClick: () => handleBatchEnable({ target: 'nonJs', enabled: true }, '已批量启用无 JS 源'),
    },
    {
      key: 'disableNonJs',
      label: '批量禁用无 JS 源',
      onClick: () => handleBatchEnable({ target: 'nonJs', enabled: false }, '已批量禁用无 JS 源'),
    },
    { type: 'divider' },
    {
      key: 'enableJs',
      label: '批量启用 JS 源',
      onClick: () => handleBatchEnable({ target: 'js', enabled: true }, '已批量启用 JS 源'),
    },
    {
      key: 'disableJs',
      label: '批量禁用 JS 源',
      onClick: () => handleBatchEnable({ target: 'js', enabled: false }, '已批量禁用 JS 源'),
    },
  ]

  const collectionSettingsContent = (
    <div className="max-w-xs">
      <Checkbox
        checked={enableOnlyNonJS}
        onChange={(e) => setEnableOnlyNonJS(e.target.checked)}
      >
        仅启用无 JS 搜索链路的书源
      </Checkbox>
      <p className="text-xs text-gray-500 mt-2 mb-0">
        勾选后导入时自动禁用搜索依赖 JS/WebView 的源，减少无效请求。
      </p>
    </div>
  )

  // 导出书源
  const handleExport = () => {
    const dataStr = JSON.stringify(rawSources.length ? rawSources : bookSources, null, 2)
    const blob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `book-sources-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  }

  const columns: any[] = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'URL',
      dataIndex: 'baseUrl',
      key: 'baseUrl',
      width: 280,
      ellipsis: true,
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean, record: BookSource) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggleEnabled(record.id, enabled)}
          checkedChildren={<CheckCircleOutlined />}
          unCheckedChildren={<CloseCircleOutlined />}
        />
      ),
    },
    {
      title: '分组',
      dataIndex: 'group',
      key: 'group',
      width: 120,
      render: (group: string) =>
        group ? <Tag>{group}</Tag> : <span style={{ color: '#999' }}>未分组</span>,
    },
    {
      title: '健康度',
      key: 'health',
      width: 100,
      render: (_: unknown, record: BookSource) => {
        const stat = sourceStats[Number(record.id)]
        if (!stat) return <span style={{ color: '#999' }}>—</span>
        const total = stat.successCount + stat.errorCount
        const rate = total > 0 ? Math.round((stat.successCount / total) * 100) : null
        return (
          <Tag color={rate === null ? 'default' : rate >= 80 ? 'green' : rate >= 50 ? 'orange' : 'red'}>
            {rate === null ? '无数据' : `${rate}%`}
          </Tag>
        )
      },
    },
    {
      title: '模式',
      dataIndex: 'mode',
      key: 'mode',
      width: 100,
      render: (mode: ParseMode) => {
        const colorMap: Record<string, string> = {
          default: 'default',
          xpath: 'blue',
          jsonpath: 'green',
          css: 'orange',
          regex: 'purple',
          js: 'magenta',
        }
        return <Tag color={colorMap[mode] ?? 'default'}>{String(mode).toUpperCase()}</Tag>
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_: any, record: BookSource) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个书源吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">书源管理</h1>
        <Space>
          <Dropdown menu={{ items: batchMenuItems }} disabled={bookSources.length === 0}>
            <Button icon={<SettingOutlined />} loading={batchLoading}>
              批量启用/禁用
            </Button>
          </Dropdown>
          <Button icon={<LinkOutlined />} onClick={() => setCollectionModalVisible(true)}>
            链接导入
          </Button>
          <Upload
            accept=".json"
            showUploadList={false}
            beforeUpload={handleImport}
          >
            <Button icon={<UploadOutlined />} loading={importLoading}>
              导入
            </Button>
          </Upload>
          <Button
            icon={<DownloadOutlined />}
            onClick={handleExport}
            disabled={bookSources.length === 0}
          >
            导出
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleAdd}
          >
            添加书源
          </Button>
        </Space>
      </div>

      {/* 搜索和筛选 */}
      <div className="flex gap-4 mb-4">
        <Search
          placeholder="搜索名称、URL、分组"
          allowClear
          value={bookSourceSearchKeyword}
          onChange={(e) => setBookSourceSearchKeyword(e.target.value)}
          style={{ width: 320 }}
          prefix={<SearchOutlined />}
        />
        <Select
          placeholder="筛选分组"
          allowClear
          value={bookSourceGroupFilter}
          onChange={(value) => setBookSourceGroupFilter(value)}
          style={{ width: 160 }}
          showSearch
          filterOption={(input, option) =>
            typeof option?.label === 'string' && option.label.toLowerCase().includes(input.toLowerCase())
          }
        >
          <Option value="">全部分组</Option>
          {groups.map((g: string) => (
            <Option key={g} value={g}>
              {g}
            </Option>
          ))}
        </Select>
        <span className="ml-auto text-gray-500 self-center">
          共 {bookSources.length} 个书源
        </span>
      </div>

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={filteredSources}
        rowKey="id"
        loading={bookSourcesLoading}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        size="middle"
        className="bg-white dark:bg-gray-800 rounded-lg"
      />

      {/* 添加/编辑模态框 */}
      <Modal
        title={editingId ? '编辑书源' : '添加书源'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false)
          form.resetFields()
        }}
        onOk={handleSave}
        confirmLoading={confirmLoading}
        width={920}
        okText={editingId ? '保存' : '添加'}
        destroyOnClose
      >
        <BookSourceFormModal form={form} />
      </Modal>

      {/* 链接导入书源合集 */}
      <Modal
        title="链接导入书源"
        open={collectionModalVisible}
        onCancel={() => {
          setCollectionModalVisible(false)
          setCollectionUrl('')
          setEnableOnlyNonJS(true)
          setCollectionSettingsOpen(false)
        }}
        onOk={handleCollectionImport}
        confirmLoading={collectionLoading}
        okText="导入"
        cancelText="取消"
      >
        <p className="text-gray-600 mb-4">
          请输入书源合集的链接地址，系统将自动解析并导入所有书源。
        </p>
        <Space.Compact block className="mb-4">
          <Input
            placeholder="https://www.yckceo.com/yuedu/shuyuans/json/id/1128.json"
            value={collectionUrl}
            onChange={(e) => setCollectionUrl(e.target.value)}
            onPressEnter={handleCollectionImport}
          />
          <Popover
            content={collectionSettingsContent}
            title="导入设置"
            trigger="click"
            open={collectionSettingsOpen}
            onOpenChange={setCollectionSettingsOpen}
            placement="bottomRight"
          >
            <Button
              icon={<SettingOutlined />}
              type={enableOnlyNonJS ? 'primary' : 'default'}
              title="导入设置"
            >
              设置
            </Button>
          </Popover>
        </Space.Compact>
        {enableOnlyNonJS && (
          <Tag color="blue">已开启：仅启用无 JS 搜索链路的书源</Tag>
        )}
      </Modal>
    </div>
  )
}
