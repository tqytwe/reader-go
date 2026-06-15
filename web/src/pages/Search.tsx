import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Input,
  Card,
  Button,
  Empty,
  Spin,
  message,
  Modal,
  Pagination,
  Typography,
  Tag,
  Divider,
  Switch,
  Tooltip,
} from 'antd'
import {
  SearchOutlined,
  ReadOutlined,
  PlusOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import { api } from '../api/client'
import { useShelfStore, useSearchStore, ShelfBook, useStore } from '../store/useStore'
import { mapStreamResult, searchBooksStream } from '../utils/searchStream'

const { Search: AntSearch } = Input
const { Text, Title } = Typography

// 搜索结果项类型
interface SearchResult {
  bookId: string
  bookName: string
  author: string
  intro: string
  sourceName: string
  bookUrl: string
  coverUrl?: string
  lastUpdate?: string
  chapterCount?: number
}

// 分页参数
interface SearchParams {
  q: string
  page: number
  pageSize: number
}

const PAGE_SIZE = 12

export default function Search() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useState<SearchParams>({
    q: '',
    page: 1,
    pageSize: PAGE_SIZE,
  })
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [total, setTotal] = useState(0)
  const [streamMode, setStreamMode] = useState(true)
  const streamCloseRef = useRef<(() => void) | null>(null)

  // 书架状态
  const { isBookInShelf } = useShelfStore()
  // 搜索历史
  const { searchHistory, addSearchHistory, clearSearchHistory } = useSearchStore()
  // 添加/移除书架操作
  const { addShelfBook, removeShelfBook } = useStore()

  // 搜索函数
  const performSearch = useCallback(async (query: string) => {
    if (!query.trim()) {
      setResults([])
      setTotal(0)
      setError(null)
      return
    }

    streamCloseRef.current?.()
    streamCloseRef.current = null

    setLoading(true)
    setError(null)
    setResults([])

    if (streamMode) {
      streamCloseRef.current = searchBooksStream(query, {
        onResult: (item) => {
          const mapped = mapStreamResult(item)
          if (!mapped.bookId) return
          setResults((prev) => {
            if (prev.some((b) => b.bookId === mapped.bookId)) return prev
            const next = [...prev, mapped as SearchResult]
            setTotal(next.length)
            return next
          })
        },
        onDone: (count) => {
          setTotal((prev) => Math.max(prev, count))
          setLoading(false)
        },
        onError: (msg) => {
          setError(msg)
          message.error(msg)
          setLoading(false)
        },
      })
      return
    }

    try {
      const response = await api.searchBooks(query) as { results?: unknown[]; total?: number }
      const list = Array.isArray(response?.results) ? response.results : []

      setResults(list.map((item: any) => ({
        bookId: item.bookKey || item.bookId || item.id || item.key,
        bookName: item.bookName || item.name || item.title,
        author: item.author || item.authorName || '未知作者',
        intro: item.intro || item.description || item.summary || '',
        sourceName: item.sourceName || item.source || item.provider || '未知来源',
        bookUrl: item.bookUrl || item.bookKey || item.url || item.key || item.bookId,
        coverUrl: item.coverUrl || item.cover || item.image,
        lastUpdate: item.lastUpdate || item.updateTime,
        chapterCount: item.chapterCount || item.chapterNum,
      })))
      setTotal(response?.total ?? list.length)
    } catch (err: any) {
      const errorMsg = err?.response?.data?.message || err?.message || '搜索失败，请重试'
      setError(errorMsg)
      message.error(errorMsg)
      setResults([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [streamMode])

  // 处理搜索提交
  const handleSearch = (value: string) => {
    if (!value.trim()) {
      message.warning('请输入搜索关键词')
      return
    }
    setSearchParams({ q: value, page: 1, pageSize: PAGE_SIZE })
    addSearchHistory(value)
    performSearch(value)
  }

  // 处理分页变化
  const handlePageChange = (page: number) => {
    setSearchParams((prev) => ({ ...prev, page }))
    performSearch(searchParams.q)
  }

  // 添加书籍到书架
  const handleAddToShelf = async (book: SearchResult) => {
    if (isBookInShelf(book.bookId)) {
      message.info('该书已在书架中')
      return
    }

    try {
      await addShelfBook({
        bookKey: book.bookId,
        name: book.bookName,
        author: book.author,
        bookUrl: book.bookUrl,
        coverUrl: book.coverUrl,
        sourceName: book.sourceName,
      } as ShelfBook)
      message.success(`已添加《${book.bookName}》到书架`)
    } catch (err: any) {
      const errorMsg = err?.response?.data?.message || '添加失败，请重试'
      message.error(errorMsg)
    }
  }

  // 从书架移除（确认弹窗）
  const handleRemoveFromShelf = (bookId: string) => {
    Modal.confirm({
      title: '确认移除',
      content: '确定要将这本书从书架移除吗？',
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        try {
          await removeShelfBook(bookId)
          message.success('已从书架移除')
        } catch (err: any) {
          const errorMsg = err?.response?.data?.message || '移除失败，请重试'
          message.error(errorMsg)
        }
      },
    })
  }

  // 清空搜索历史
  const handleClearHistory = () => {
    Modal.confirm({
      title: '清空历史',
      content: '确定要清空所有搜索历史吗？',
      onOk: clearSearchHistory,
    })
  }

  // 点击历史项重新搜索
  const handleHistoryClick = (item: { keyword: string }) => {
    setSearchParams({ q: item.keyword, page: 1, pageSize: PAGE_SIZE })
    performSearch(item.keyword)
  }

  useEffect(() => {
    return () => streamCloseRef.current?.()
  }, [])

  // 监听搜索参数变化
  useEffect(() => {
    if (searchParams.q) {
      performSearch(searchParams.q)
    }
  }, [searchParams.q]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="p-4 md:p-6 max-w-7xl mx-auto">
      {/* 搜索框区域 */}
      <div className="mb-6">
        <div className="flex flex-col md:flex-row gap-3">
          <AntSearch
            placeholder="搜索书名、作者..."
            size="large"
            allowClear
            onSearch={handleSearch}
            onChange={(e) => setSearchParams(prev => ({ ...prev, q: e.target.value }))}
            enterButton={
              <Button type="primary" icon={<SearchOutlined />}>
                搜索
              </Button>
            }
            className="flex-1"
          />
          <Tooltip title="开启后通过 SSE 逐条显示结果，无需等待全部书源完成">
            <div className="flex items-center gap-2 whitespace-nowrap px-2">
              <Switch checked={streamMode} onChange={setStreamMode} size="small" />
              <Text type="secondary" className="text-sm">流式</Text>
            </div>
          </Tooltip>
        </div>

        {/* 搜索历史 */}
        {searchHistory.length > 0 && (
          <div className="mt-3 flex items-center gap-2 flex-wrap">
            <Text type="secondary" className="text-sm">最近搜索:</Text>
            {searchHistory.slice(0, 8).map((item, idx) => (
              <Tag
                key={idx}
                closable={false}
                onClick={() => handleHistoryClick(item)}
                className="cursor-pointer hover:bg-blue-50"
              >
                {item.keyword}
              </Tag>
            ))}
            <Button
              type="link"
              size="small"
              onClick={handleClearHistory}
              className="text-gray-400"
            >
              清空
            </Button>
          </div>
        )}
      </div>

      {/* 加载状态 */}
      {loading && (
        <div className="flex justify-center py-12">
          <Spin size="large" tip="搜索中..." />
        </div>
      )}

      {/* 错误状态 */}
      {error && !loading && (
        <div className="text-center py-12">
          <Empty
            description={
              <span>
                <span className="text-red-500">{error}</span>
              </span>
            }
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      )}

      {/* 未搜索状态 */}
      {!loading && !error && results.length === 0 && searchParams.q === '' && (
        <div className="text-center py-20">
          <ReadOutlined className="text-6xl text-gray-300 mb-4" />
          <Title level={3} className="text-gray-500">开始搜索吧</Title>
          <Text type="secondary">输入书名或作者，找到你想读的书</Text>
        </div>
      )}

      {/* 无结果状态 */}
      {!loading && !error && results.length === 0 && searchParams.q !== '' && (
        <div className="text-center py-20">
          <Empty
            description={`未找到与 "${searchParams.q}" 相关的书籍`}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      )}

      {/* 搜索结果列表 */}
      {!loading && !error && results.length > 0 && (
        <>
          <div className="flex items-center justify-between mb-4">
            <Text className="text-gray-500">
              找到 {total} 条结果 (第 {searchParams.page} 页)
            </Text>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {results.map((book) => {
              const inShelf = isBookInShelf(book.bookId)
              return (
                <Card
                  key={book.bookId}
                  hoverable
                  className="overflow-hidden"
                  cover={
                    book.coverUrl ? (
                      <div className="aspect-[2/3] bg-gradient-to-br from-blue-100 to-purple-100 flex items-center justify-center overflow-hidden">
                        <img
                          src={book.coverUrl}
                          alt={book.bookName}
                          className="w-full h-full object-cover"
                          onError={(e) => {
                            e.currentTarget.style.display = 'none'
                          }}
                        />
                      </div>
                    ) : (
                      <div className="aspect-[2/3] bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                        <ReadOutlined className="text-white text-5xl opacity-50" />
                      </div>
                    )
                  }
                >
                  <Card.Meta
                    title={
                      <div className="line-clamp-1" title={book.bookName}>
                        {book.bookName}
                      </div>
                    }
                    description={
                      <div className="mt-2">
                        <Text type="secondary" className="text-sm">
                          {book.author}
                        </Text>
                        {book.sourceName && (
                          <div className="mt-1">
                            <Tag color="blue">{book.sourceName}</Tag>
                          </div>
                        )}
                        {book.intro && (
                          <Text
                            className="mt-2 line-clamp-2 text-xs text-gray-500 block"
                            type="secondary"
                          >
                            {book.intro}
                          </Text>
                        )}
                      </div>
                    }
                  />
                  <Divider className="my-3" />
                  <div className="flex gap-2">
                    {inShelf ? (
                      <>
                        <Button
                          type="primary"
                          icon={<CheckCircleOutlined />}
                          size="small"
                          className="flex-1"
                          disabled
                        >
                          已在书架
                        </Button>
                        <Button
                          danger
                          size="small"
                          onClick={() => handleRemoveFromShelf(book.bookId)}
                        >
                          移除
                        </Button>
                      </>
                    ) : (
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        size="small"
                        className="flex-1"
                        onClick={() => handleAddToShelf(book)}
                      >
                        加入书架
                      </Button>
                    )}
                    <Button
                      size="small"
                      icon={<ReadOutlined />}
                      onClick={() => {
                        if (!book.bookId?.trim()) {
                          message.warning('书籍标识无效，无法打开阅读器')
                          return
                        }
                        navigate(`/reader/${encodeURIComponent(book.bookId)}`)
                      }}
                    >
                      阅读
                    </Button>
                  </div>
                </Card>
              )
            })}
          </div>

          {/* 分页 */}
          <div className="flex justify-center mt-8">
            <Pagination
              current={searchParams.page}
              total={total}
              pageSize={PAGE_SIZE}
              onChange={handlePageChange}
              showSizeChanger={false}
              showQuickJumper
              showTotal={(t) => `共 ${t} 条`}
            />
          </div>
        </>
      )}
    </div>
  )
}
