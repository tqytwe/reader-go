import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Typography, Badge, Empty } from 'antd'
import {
  BookOutlined,
  ReadOutlined,
  CheckCircleOutlined,
  SettingOutlined,
  SearchOutlined,
  CompassOutlined,
  GlobalOutlined,
  FolderOpenOutlined,
  CloudSyncOutlined,
  FilterOutlined,
  EyeOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import { useShelfStore } from '../store/useStore'
import { useStore } from '../store/useStore'

const { Title, Text, Paragraph } = Typography

export default function Home() {
  const navigate = useNavigate()
  const { books: rawBooks } = useShelfStore()
  const { bookSources } = useStore()
  const books = Array.isArray(rawBooks) ? rawBooks : []

  // 统计信息
  const stats = useMemo(() => ({
    total: books.length,
    reading: books.filter((b) => (b.progress ?? 0) > 0 && (b.progress ?? 0) < 100).length,
    finished: books.filter((b) => (b.progress ?? 0) >= 100).length,
    sources: bookSources.length,
    enabledSources: bookSources.filter((s) => s.enabled).length,
  }), [books, bookSources])

  // 继续阅读（最近阅读的书籍，按 lastReadAt 排序）
  const continueReading = useMemo(() => {
    return [...books]
      .filter((b) => (b.progress ?? 0) > 0)
      .sort((a, b) => {
        const aTime = new Date(a.lastReadAt || a.createdAt || 0).getTime()
        const bTime = new Date(b.lastReadAt || b.createdAt || 0).getTime()
        return bTime - aTime
      })
      .slice(0, 8)
  }, [books])

  // 最近添加
  const recentAdded = useMemo(() => {
    return [...books]
      .sort((a, b) => {
        const aTime = new Date(a.createdAt || 0).getTime()
        const bTime = new Date(b.createdAt || 0).getTime()
        return bTime - aTime
      })
      .slice(0, 6)
  }, [books])

  const today = new Date()
  const dateStr = `${today.getFullYear()}年${today.getMonth() + 1}月${today.getDate()}日`
  const weekDays = ['日', '一', '二', '三', '四', '五', '六']
  const dayStr = `星期${weekDays[today.getDay()]}`

  // 快捷入口
  const quickLinks = [
    { icon: <SearchOutlined />, label: '搜索书籍', color: '#1677ff', path: '/search' },
    { icon: <CompassOutlined />, label: '书海浏览', color: '#52c41a', path: '/explore' },
    { icon: <BookOutlined />, label: '我的书架', color: '#fa8c16', path: '/bookshelf' },
    { icon: <GlobalOutlined />, label: 'RSS 订阅', color: '#eb2f96', path: '/rss' },
    { icon: <SettingOutlined />, label: '书源管理', color: '#722ed1', path: '/booksource' },
    { icon: <FolderOpenOutlined />, label: '本地书籍', color: '#13c2c2', path: '/localBooks' },
    { icon: <FilterOutlined />, label: '替换规则', color: '#faad14', path: '/replaceRules' },
    { icon: <CloudSyncOutlined />, label: '数据同步', color: '#2f54eb', path: '/sync' },
  ]

  return (
    <div className="p-4 md:p-6 max-w-6xl mx-auto">
      {/* 欢迎区域 */}
      <div className="mb-6">
        <Title level={3} style={{ marginBottom: 4 }}>
          {books.length > 0 ? '欢迎回来' : '开始阅读'}
        </Title>
        <Text style={{ color: 'var(--app-muted)' }}>
          {dateStr} {dayStr}
          {books.length > 0 && ` · 书架上有 ${stats.total} 本书`}
        </Text>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <Card className="text-center py-2" hoverable onClick={() => navigate('/bookshelf')}>
          <div className="flex items-center justify-center gap-2 mb-1">
            <BookOutlined style={{ fontSize: 20, color: '#1677ff' }} />
            <span className="text-2xl font-bold" style={{ color: '#1677ff' }}>{stats.total}</span>
          </div>
          <Text type="secondary" className="text-sm">书架书籍</Text>
        </Card>
        <Card className="text-center py-2" hoverable onClick={() => navigate('/bookshelf')}>
          <div className="flex items-center justify-center gap-2 mb-1">
            <EyeOutlined style={{ fontSize: 20, color: '#faad14' }} />
            <span className="text-2xl font-bold" style={{ color: '#faad14' }}>{stats.reading}</span>
          </div>
          <Text type="secondary" className="text-sm">阅读中</Text>
        </Card>
        <Card className="text-center py-2" hoverable onClick={() => navigate('/bookshelf')}>
          <div className="flex items-center justify-center gap-2 mb-1">
            <CheckCircleOutlined style={{ fontSize: 20, color: '#52c41a' }} />
            <span className="text-2xl font-bold" style={{ color: '#52c41a' }}>{stats.finished}</span>
          </div>
          <Text type="secondary" className="text-sm">已读完</Text>
        </Card>
        <Card className="text-center py-2" hoverable onClick={() => navigate('/booksource')}>
          <div className="flex items-center justify-center gap-2 mb-1">
            <SettingOutlined style={{ fontSize: 20, color: '#722ed1' }} />
            <span className="text-2xl font-bold" style={{ color: '#722ed1' }}>{stats.enabledSources}</span>
          </div>
          <Text type="secondary" className="text-sm">启用书源 / {stats.sources}</Text>
        </Card>
      </div>

      {/* 继续阅读 */}
      {continueReading.length > 0 && (
        <div className="mb-6">
          <div className="flex items-center justify-between mb-3">
            <Title level={5} style={{ marginBottom: 0 }}>
              <ClockCircleOutlined className="mr-2" style={{ color: '#faad14' }} />
              继续阅读
            </Title>
            <a onClick={() => navigate('/bookshelf')} className="text-sm">
              查看全部 →
            </a>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {continueReading.map((book) => (
              <Card
                key={book.bookKey}
                hoverable
                size="small"
                className="overflow-hidden cursor-pointer"
                onClick={() => navigate(`/reader/${encodeURIComponent(book.bookKey)}`)}
                cover={
                  <div className="aspect-[2/3] bg-gradient-to-br from-blue-100 to-purple-100 relative overflow-hidden">
                    {book.coverUrl ? (
                      <img
                        src={book.coverUrl}
                        alt={book.name}
                        className="w-full h-full object-cover"
                        onError={(e) => { e.currentTarget.style.display = 'none' }}
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <ReadOutlined className="text-3xl text-gray-300" />
                      </div>
                    )}
                    {/* 进度条 */}
                    <div className="absolute bottom-0 left-0 right-0 h-1 bg-gray-200/50">
                      <div
                        className="h-full bg-blue-500 transition-all"
                        style={{ width: `${Math.min(book.progress ?? 0, 100)}%` }}
                      />
                    </div>
                    {/* 进度标签 */}
                    <Badge
                      className="absolute top-2 right-2"
                      count={`${Math.round(book.progress ?? 0)}%`}
                      style={{ backgroundColor: '#1677ff' }}
                    />
                  </div>
                }
              >
                <Card.Meta
                  title={
                    <span className="text-sm line-clamp-1" title={book.name}>
                      {book.name}
                    </span>
                  }
                  description={
                    <span className="text-xs text-gray-500 line-clamp-1">
                      {book.currentChapter || book.author}
                    </span>
                  }
                />
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* 最近添加（当没有阅读记录时显示） */}
      {continueReading.length === 0 && recentAdded.length > 0 && (
        <div className="mb-6">
          <div className="flex items-center justify-between mb-3">
            <Title level={5} style={{ marginBottom: 0 }}>
              <BookOutlined className="mr-2" style={{ color: '#1677ff' }} />
              最近添加
            </Title>
            <a onClick={() => navigate('/bookshelf')} className="text-sm">
              查看全部 →
            </a>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {recentAdded.map((book) => (
              <Card
                key={book.bookKey}
                hoverable
                size="small"
                className="overflow-hidden cursor-pointer"
                onClick={() => navigate(`/reader/${encodeURIComponent(book.bookKey)}`)}
                cover={
                  <div className="aspect-[2/3] bg-gradient-to-br from-blue-100 to-purple-100 relative overflow-hidden">
                    {book.coverUrl ? (
                      <img
                        src={book.coverUrl}
                        alt={book.name}
                        className="w-full h-full object-cover"
                        onError={(e) => { e.currentTarget.style.display = 'none' }}
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <ReadOutlined className="text-3xl text-gray-300" />
                      </div>
                    )}
                  </div>
                }
              >
                <Card.Meta
                  title={<span className="text-xs line-clamp-1">{book.name}</span>}
                  description={<span className="text-xs text-gray-400">{book.author}</span>}
                />
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* 快捷入口 */}
      <div className="mb-6">
        <Title level={5} className="mb-3">快捷入口</Title>
        <div className="grid grid-cols-4 md:grid-cols-8 gap-3">
          {quickLinks.map((link) => (
            <Card
              key={link.path}
              hoverable
              size="small"
              className="text-center cursor-pointer"
              onClick={() => navigate(link.path)}
              styles={{ body: { padding: '12px 8px' } }}
            >
              <div style={{ fontSize: 24, color: link.color, marginBottom: 4 }}>
                {link.icon}
              </div>
              <Text className="text-xs block">{link.label}</Text>
            </Card>
          ))}
        </div>
      </div>

      {/* 空书架引导 */}
      {books.length === 0 && (
        <Card className="text-center py-8">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <div>
                <Paragraph type="secondary">书架是空的，开始探索吧</Paragraph>
              </div>
            }
          >
            <div className="flex gap-3 justify-center">
              <a onClick={() => navigate('/search')} className="ant-btn ant-btn-primary">
                搜索书籍
              </a>
              <a onClick={() => navigate('/explore')} className="ant-btn">
                浏览书海
              </a>
            </div>
          </Empty>
        </Card>
      )}
    </div>
  )
}
