import { useState, useMemo, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Input,
  Card,
  Button,
  Empty,
  message,
  Modal,
  Typography,
  Tag,
  Dropdown,
  Divider,
  Badge,
  Spin,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  ReadOutlined,
  DeleteOutlined,
  SearchOutlined,
  MoreOutlined,
  ClockCircleOutlined,
  BookOutlined,
  EyeOutlined,
  SwapOutlined,
} from '@ant-design/icons'
import { api } from '../api/client'
import { useShelfStore, type ShelfBook } from '../store/useStore'
import type { BookAlternateCandidate } from '../types/booksource'

const { Search } = Input
const { Text, Title } = Typography

// 分组类型
type GroupType = 'all' | 'unread' | 'reading' | 'finished'

export default function Bookshelf() {
  const navigate = useNavigate()
  const { books: rawBooks, removeBook, updateBook } = useShelfStore()
  const books = Array.isArray(rawBooks) ? rawBooks : []
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [currentGroup, setCurrentGroup] = useState<GroupType>('all')
  const [sortBy, setSortBy] = useState<'createdAt' | 'lastReadAt' | 'name'>('lastReadAt')
  const [switchModalOpen, setSwitchModalOpen] = useState(false)
  const [switchTarget, setSwitchTarget] = useState<ShelfBook | null>(null)
  const [alternates, setAlternates] = useState<BookAlternateCandidate[]>([])
  const [alternatesLoading, setAlternatesLoading] = useState(false)
  const [switching, setSwitching] = useState(false)

  // 从 store 加载书架数据
  useEffect(() => {
    const loadShelf = async () => {
      setLoading(true)
      try {
        const data = await api.getShelf() as { books?: ShelfBook[] }
        useShelfStore.getState().setBooks(data?.books ?? [])
      } catch (err) {
        console.error('加载书架失败:', err)
        message.error('加载书架失败，请刷新页面重试')
      } finally {
        setLoading(false)
      }
    }
    loadShelf()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 过滤和排序书架书籍
  const filteredBooks = useMemo(() => {
    let result = [...books]

    // 按分组筛选
    if (currentGroup === 'unread') {
      result = result.filter((b) => (b.progress ?? 0) === 0)
    } else if (currentGroup === 'reading') {
      result = result.filter((b) => (b.progress ?? 0) > 0 && (b.progress ?? 0) < 100)
    } else if (currentGroup === 'finished') {
      result = result.filter((b) => (b.progress ?? 0) >= 100)
    }

    // 按搜索关键词筛选
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(
        (b) =>
          b.name.toLowerCase().includes(q) ||
          b.author.toLowerCase().includes(q) ||
          (b.sourceName || '').toLowerCase().includes(q)
      )
    }

    // 排序
    result.sort((a, b) => {
      let comparison = 0
      if (sortBy === 'name') {
        comparison = a.name.localeCompare(b.name, 'zh')
      } else if (sortBy === 'createdAt') {
        const aTime = new Date(a.createdAt || 0).getTime()
        const bTime = new Date(b.createdAt || 0).getTime()
        comparison = aTime - bTime
      } else if (sortBy === 'lastReadAt') {
        const aTime = new Date(a.lastReadAt || 0).getTime()
        const bTime = new Date(b.lastReadAt || 0).getTime()
        comparison = aTime - bTime
      }
      return comparison
    })

    return result
  }, [books, currentGroup, searchQuery, sortBy])

  // 统计信息
  const stats = useMemo(() => ({
    total: books.length,
    unread: books.filter((b) => (b.progress ?? 0) === 0).length,
    reading: books.filter((b) => (b.progress ?? 0) > 0 && (b.progress ?? 0) < 100).length,
    finished: books.filter((b) => (b.progress ?? 0) >= 100).length,
  }), [books])

  // 移除书架
  const handleRemove = (bookKey: string, bookName: string) => {
    Modal.confirm({
      title: '确认移除',
      content: `确定要将《${bookName}》从书架移除吗？`,
      okText: '确认移除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await removeBook(bookKey)
          message.success('已从书架移除')
        } catch (err: any) {
          const errorMsg = err?.response?.data?.message || '移除失败，请重试'
          message.error(errorMsg)
        }
      },
    })
  }

  // 进入阅读器（SPA 路由，避免整页刷新导致报错）
  const handleGoReader = (bookKey: string) => {
    if (!bookKey?.trim()) {
      message.warning('书籍标识无效，无法打开阅读器')
      return
    }
    navigate(`/reader/${encodeURIComponent(bookKey)}`)
  }

  const openSwitchSource = async (book: ShelfBook) => {
    setSwitchTarget(book)
    setSwitchModalOpen(true)
    setAlternatesLoading(true)
    try {
      const res = (await api.getBookAlternates(book.bookKey)) as {
        candidates?: BookAlternateCandidate[]
      }
      setAlternates(res.candidates ?? [])
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '获取候选书源失败')
      setAlternates([])
    } finally {
      setAlternatesLoading(false)
    }
  }

  const handleSwitchSource = async (candidate: BookAlternateCandidate) => {
    if (!switchTarget?.id) {
      message.error('书架记录无效，无法换源')
      return
    }
    setSwitching(true)
    try {
      await api.updateShelfBook(String(switchTarget.id), {
        ...switchTarget,
        bookKey: candidate.bookKey,
        sourceId: candidate.sourceId,
        sourceName: candidate.sourceName,
      })
      updateBook(switchTarget.bookKey, {
        bookKey: candidate.bookKey,
        sourceId: candidate.sourceId,
        sourceName: candidate.sourceName,
      })
      message.success(`已切换到「${candidate.sourceName}」`)
      setSwitchModalOpen(false)
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '换源失败')
    } finally {
      setSwitching(false)
    }
  }

  // 批量操作
  const handleClearFinished = async () => {
    const finishedBooks = books.filter((b) => (b.progress ?? 0) >= 100)
    if (finishedBooks.length === 0) {
      message.info('没有已读完的书籍')
      return
    }
    Modal.confirm({
      title: '清空已读完',
      content: `将移除 ${finishedBooks.length} 本已读完的书籍，确认吗？`,
      okText: '确认',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await Promise.all(finishedBooks.map((b) => removeBook(b.bookKey)))
          message.success(`已移除 ${finishedBooks.length} 本书`)
        } catch (err) {
          message.error('批量移除失败')
        }
      },
    })
  }

  // 导出书架
  const handleExport = () => {
    const exportData = JSON.stringify(books, null, 2)
    const blob = new Blob([exportData], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `shelf-export-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  }

  // 排序菜单项
  const sortMenuItems: MenuProps['items'] = [
    { key: 'lastReadAt', label: '最近阅读', onClick: () => setSortBy('lastReadAt') },
    { key: 'createdAt', label: '添加时间', onClick: () => setSortBy('createdAt') },
    { key: 'name', label: '书名', onClick: () => setSortBy('name') },
  ]

  // 合并所有菜单项
  const allMenuItems: MenuProps['items'] = [
    {
      key: 'sort-group',
      label: '排序方式',
      children: sortMenuItems,
    },
    { type: 'divider' },
    { key: 'clearFinished', label: '清空已读完', icon: <DeleteOutlined />, danger: true, onClick: handleClearFinished },
    { key: 'export', label: '导出书架', icon: <ReadOutlined />, onClick: handleExport },
  ]

  return (
    <div className="p-4 md:p-6 max-w-7xl mx-auto">
      {/* 顶部操作栏 */}
      <div className="flex flex-col md:flex-row gap-4 mb-6">
        <div className="flex-1">
          <Search
            placeholder="搜索书架..."
            size="large"
            allowClear
            onSearch={setSearchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            prefix={<SearchOutlined className="text-gray-400" />}
            className="w-full"
          />
        </div>
        <div className="flex gap-2">
          <Button
            type={currentGroup === 'all' ? 'primary' : 'default'}
            onClick={() => setCurrentGroup('all')}
            icon={<BookOutlined />}
          >
            全部
          </Button>
          <Button
            type={currentGroup === 'unread' ? 'primary' : 'default'}
            onClick={() => setCurrentGroup('unread')}
            icon={<ReadOutlined />}
          >
            未读
          </Button>
          <Button
            type={currentGroup === 'reading' ? 'primary' : 'default'}
            onClick={() => setCurrentGroup('reading')}
            icon={<EyeOutlined />}
          >
            阅读中
          </Button>
          <Button
            type={currentGroup === 'finished' ? 'primary' : 'default'}
            onClick={() => setCurrentGroup('finished')}
            icon={<ClockCircleOutlined />}
          >
            已读完
          </Button>
        </div>
        <Dropdown
          menu={{
            items: allMenuItems,
          }}
          placement="bottomRight"
        >
          <Button icon={<MoreOutlined />}>更多</Button>
        </Dropdown>
      </div>

      {/* 统计信息 */}
      <div className="flex gap-4 mb-6">
        <Card className="flex-1 text-center py-3">
          <div className="text-2xl font-bold text-blue-600">{stats.total}</div>
          <div className="text-gray-500 text-sm">总数</div>
        </Card>
        <Card className="flex-1 text-center py-3">
          <div className="text-2xl font-bold text-gray-600">{stats.unread}</div>
          <div className="text-gray-500 text-sm">未读</div>
        </Card>
        <Card className="flex-1 text-center py-3">
          <div className="text-2xl font-bold text-yellow-600">{stats.reading}</div>
          <div className="text-gray-500 text-sm">阅读中</div>
        </Card>
        <Card className="flex-1 text-center py-3">
          <div className="text-2xl font-bold text-green-600">{stats.finished}</div>
          <div className="text-gray-500 text-sm">已读完</div>
        </Card>
      </div>

      {/* 加载状态 */}
      {loading && (
        <div className="flex justify-center py-20">
          <Spin size="large" tip="加载中..." />
        </div>
      )}

      {/* 空书架 */}
      {!loading && books.length === 0 && (
        <div className="text-center py-20">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <div>
                <p className="text-gray-500 mb-4">书架是空的</p>
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={() => navigate('/search')}
                >
                  去搜索
                </Button>
              </div>
            }
          />
        </div>
      )}

      {/* 搜索无结果 */}
      {!loading && books.length > 0 && filteredBooks.length === 0 && (
        <div className="text-center py-20">
          <Empty
            description={`在书架中未找到与 "${searchQuery}" 相关的书籍`}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      )}

      {/* 书籍网格 */}
      {!loading && filteredBooks.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
          {filteredBooks.map((book) => (
            <Card
              key={book.bookKey}
              hoverable
              className="overflow-hidden group"
              cover={
                <div
                  className="aspect-[2/3] bg-gradient-to-br from-blue-100 to-purple-100 cursor-pointer relative overflow-hidden"
                  onClick={() => handleGoReader(book.bookKey)}
                >
                  {book.coverUrl ? (
                    <img
                      src={book.coverUrl}
                      alt={book.name}
                      className="w-full h-full object-cover transition-transform group-hover:scale-105"
                      onError={(e) => {
                        e.currentTarget.style.display = 'none'
                      }}
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <ReadOutlined className="text-4xl text-gray-300" />
                    </div>
                  )}
                  {/* 进度覆盖层 */}
                  {(book.progress ?? 0) > 0 && (
                    <div
                      className="absolute bottom-0 left-0 right-0 h-1 bg-gray-200"
                      style={{ opacity: 0.8 }}
                    >
                      <div
                        className="h-full bg-blue-500 transition-all"
                        style={{ width: `${Math.min(book.progress ?? 0, 100)}%` }}
                      />
                    </div>
                  )}
                  {/* 进度标签 */}
                  {(book.progress ?? 0) > 0 && (
                    <Badge
                      className="absolute top-2 right-2"
                      count={`${Math.round(book.progress ?? 0)}%`}
                      style={{ backgroundColor: (book.progress ?? 0) >= 100 ? '#52c41a' : '#1890ff' }}
                    />
                  )}
                </div>
              }
            >
              <div
                className="cursor-pointer"
                onClick={() => handleGoReader(book.bookKey)}
              >
                <Title level={5} className="line-clamp-1 mb-1" style={{ fontSize: '14px' }}>
                  {book.name}
                </Title>
                <Text type="secondary" className="text-xs block mb-2">
                  {book.author}
                </Text>
                {book.currentChapter && (
                  <Tag color="blue" className="mb-2">
                    {book.currentChapter}
                  </Tag>
                )}
              </div>
              <Divider className="my-2" />
              <div className="flex gap-1">
                <Button
                  size="small"
                  type="primary"
                  className="flex-1"
                  icon={<EyeOutlined />}
                  onClick={() => handleGoReader(book.bookKey)}
                >
                  阅读
                </Button>
                <Button
                  size="small"
                  icon={<SwapOutlined />}
                  title="换源"
                  onClick={(e) => {
                    e.stopPropagation()
                    openSwitchSource(book)
                  }}
                />
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleRemove(book.bookKey, book.name)
                  }}
                />
              </div>
            </Card>
          ))}
        </div>
      )}

      <Modal
        title={switchTarget ? `换源 —《${switchTarget.name}》` : '换源'}
        open={switchModalOpen}
        onCancel={() => setSwitchModalOpen(false)}
        footer={null}
      >
        {alternatesLoading ? (
          <div className="py-8 text-center">
            <Spin tip="加载候选书源..." />
          </div>
        ) : alternates.length === 0 ? (
          <Empty description="暂无其他书源候选" />
        ) : (
          <div className="space-y-2">
            <Text type="secondary" className="block mb-3">
              系统会在其他书源中搜索同名书籍，并按章节名对齐阅读进度（需已有 currentChapter）
            </Text>
            {alternates.map((c) => (
              <Card key={`${c.sourceId}-${c.bookKey}`} size="small" className="hover:border-blue-400">
                <div className="flex items-center justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <Tag color="blue">{c.sourceName}</Tag>
                    {c.name && <Text className="ml-2">{c.name}</Text>}
                    {c.author && <Text type="secondary" className="text-xs block">{c.author}</Text>}
                    <div className="mt-1 flex flex-wrap gap-1">
                      {c.matchScore != null && (
                        <Tag color={c.matchScore >= 0.8 ? 'green' : 'orange'}>
                          匹配 {Math.round(c.matchScore * 100)}%
                        </Tag>
                      )}
                      {c.chapterScore != null && c.chapterScore > 0 && (
                        <Tag color={c.chapterScore >= 0.8 ? 'cyan' : 'default'}>
                          章节 {Math.round(c.chapterScore * 100)}%
                          {c.chapterIndex != null && c.chapterIndex >= 0 ? ` · 第${c.chapterIndex + 1}章` : ''}
                        </Tag>
                      )}
                    </div>
                  </div>
                  <Button
                    type="primary"
                    size="small"
                    loading={switching}
                    onClick={() => handleSwitchSource(c)}
                  >
                    切换
                  </Button>
                </div>
              </Card>
            ))}
          </div>
        )}
      </Modal>
    </div>
  )
}
