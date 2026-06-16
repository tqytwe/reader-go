import { useEffect, useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Upload, Table, Button, message, Space, Popconfirm, Tag,
  Input, Empty, Modal, Tooltip,
} from 'antd'
import {
  UploadOutlined, ReadOutlined, DeleteOutlined, SearchOutlined,
  FileTextOutlined, BookOutlined, FileOutlined, InboxOutlined,
} from '@ant-design/icons'
import { api, type LocalBook } from '../api/client'

const { Dragger } = Upload

const FORMAT_COLORS: Record<string, string> = {
  txt: 'blue',
  epub: 'green',
  cbz: 'orange',
  pdf: 'red',
}

const FORMAT_ICONS: Record<string, React.ReactNode> = {
  txt: <FileTextOutlined />,
  epub: <BookOutlined />,
  cbz: <FileOutlined />,
  pdf: <FileTextOutlined />,
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

export default function LocalBooks() {
  const navigate = useNavigate()
  const [books, setBooks] = useState<LocalBook[]>([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])

  const load = async () => {
    setLoading(true)
    try {
      const data: unknown = await api.listLocalBooks()
      setBooks(Array.isArray(data) ? (data as LocalBook[]) : [])
    } catch {
      message.error('加载本地书籍失败')
      setBooks([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])

  // 过滤后的书籍
  const filteredBooks = useMemo(() => {
    if (!searchQuery.trim()) return books
    const q = searchQuery.toLowerCase()
    return books.filter(b =>
      b.name.toLowerCase().includes(q) ||
      b.author.toLowerCase().includes(q) ||
      b.format.toLowerCase().includes(q)
    )
  }, [books, searchQuery])

  const handleDelete = async (id: string) => {
    try {
      await api.deleteLocalBook(id)
      message.success('已删除')
      await load()
    } catch {
      message.error('删除失败')
    }
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的书籍')
      return
    }
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 本书籍吗？`,
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await Promise.all(selectedRowKeys.map(id => api.deleteLocalBook(String(id))))
          setSelectedRowKeys([])
          message.success(`已删除 ${selectedRowKeys.length} 本书籍`)
          await load()
        } catch {
          message.error('批量删除失败')
        }
      },
    })
  }

  const uploadProps = {
    showUploadList: false,
    beforeUpload: async (file: File) => {
      setUploading(true)
      try {
        await api.uploadLocalBook(file)
        message.success(`《${file.name}》上传成功`)
        await load()
      } catch (e: unknown) {
        message.error(e instanceof Error ? e.message : '上传失败')
      } finally {
        setUploading(false)
      }
      return false
    },
    accept: '.txt,.epub,.cbz,.pdf',
    multiple: true,
  }

  const columns = [
    {
      title: '书名',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
      render: (name: string, record: LocalBook) => (
        <div className="flex items-center gap-2">
          <span style={{ color: 'var(--app-accent, #1677ff)' }}>
            {FORMAT_ICONS[record.format.toLowerCase()] || <FileTextOutlined />}
          </span>
          <span className="font-medium">{name}</span>
        </div>
      ),
    },
    {
      title: '作者',
      dataIndex: 'author',
      key: 'author',
      width: 150,
      ellipsis: true,
      render: (v: string) => v || <span className="text-gray-400">未知</span>,
    },
    {
      title: '格式',
      dataIndex: 'format',
      key: 'format',
      width: 80,
      render: (format: string) => (
        <Tag color={FORMAT_COLORS[format.toLowerCase()] || 'default'}>
          {format.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '大小',
      dataIndex: 'fileSize',
      key: 'fileSize',
      width: 100,
      render: (size: number) => size ? formatFileSize(size) : '-',
    },
    {
      title: '章节',
      dataIndex: 'chapters',
      key: 'chapters',
      width: 80,
      render: (chapters: any[]) => chapters?.length || '-',
    },
    {
      title: '上传时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render: (_: unknown, record: LocalBook) => (
        <Space size="small">
          <Tooltip title="阅读">
            <Button
              type="link"
              size="small"
              icon={<ReadOutlined />}
              onClick={() => navigate(`/reader/local-${record.id}`)}
            >
              阅读
            </Button>
          </Tooltip>
          <Popconfirm
            title="确定要删除这本书吗？"
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
        <h1 className="text-2xl font-bold">本地书籍</h1>
        <Space>
          <Button
            danger
            icon={<DeleteOutlined />}
            onClick={handleBatchDelete}
            disabled={selectedRowKeys.length === 0}
          >
            批量删除 {selectedRowKeys.length > 0 && `(${selectedRowKeys.length})`}
          </Button>
          <Upload {...uploadProps}>
            <Button icon={<UploadOutlined />} loading={uploading}>
              上传书籍
            </Button>
          </Upload>
        </Space>
      </div>

      {/* 搜索 */}
      <div className="flex gap-4 mb-4">
        <Input
          placeholder="搜索书名、作者..."
          prefix={<SearchOutlined />}
          allowClear
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{ width: 280 }}
        />
        <span className="ml-auto text-gray-500 self-center text-sm">
          共 {books.length} 本{filteredBooks.length !== books.length && `，显示 ${filteredBooks.length} 本`}
        </span>
      </div>

      {/* 空状态 + 拖拽上传 */}
      {books.length === 0 && !loading ? (
        <div className="py-8">
          <Dragger
            {...uploadProps}
            className="bg-transparent"
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined style={{ fontSize: 48, color: 'var(--app-accent, #1677ff)' }} />
            </p>
            <p className="ant-upload-text text-lg">点击或拖拽文件到此区域上传</p>
            <p className="ant-upload-hint">
              支持 TXT、EPUB、CBZ、PDF 格式的本地书籍文件
            </p>
          </Dragger>
        </div>
      ) : filteredBooks.length === 0 && !loading ? (
        <Empty description={`未找到与 "${searchQuery}" 相关的书籍`} />
      ) : (
        <Table
          loading={loading}
          rowKey="id"
          dataSource={filteredBooks}
          columns={columns}
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
    </div>
  )
}
