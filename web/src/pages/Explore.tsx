import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Select, Card, Spin, message, Empty, Button, Tag, Typography, Pagination } from 'antd'
import { PlusOutlined, ReadOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import { useShelfStore, useStore, type ShelfBook } from '../store/useStore'

const { Text, Paragraph } = Typography

interface ExploreTab {
  title: string
  url: string
}

interface ExploreBook {
  name: string
  author?: string
  intro?: string
  coverUrl?: string
  bookUrl?: string
  bookKey: string
  sourceName?: string
}

interface ExploreResponse {
  tab?: string
  tabs?: ExploreTab[]
  books?: ExploreBook[]
  items?: ExploreBook[]
  page?: number
  pageSize?: number
  hasMore?: boolean
  total?: number
}

export default function Explore() {
  const navigate = useNavigate()
  const [sources, setSources] = useState<{ id: number; name: string; exploreUrl?: string }[]>([])
  const [sourceId, setSourceId] = useState<number>()
  const [tab, setTab] = useState('')
  const [tabs, setTabs] = useState<ExploreTab[]>([])
  const [loading, setLoading] = useState(false)
  const [books, setBooks] = useState<ExploreBook[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(30)
  const [total, setTotal] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const { isBookInShelf } = useShelfStore()
  const { addShelfBook } = useStore()
  const requestIdRef = useRef(0)

  useEffect(() => {
    api.getBookSources().then((data: unknown) => {
      const arr = Array.isArray(data) ? data : []
      const list = (arr as { id: number; name: string; exploreUrl?: string }[]).filter((s) => s.exploreUrl)
      setSources(list)
      if (list[0]) setSourceId(list[0].id)
    }).catch(() => {
      setSources([])
    })
  }, [])

  const tabOptions = useMemo(() => {
    const valid = (tabs || []).filter((t) => t.url?.trim())
    if (valid.length > 0) {
      return valid.map((t) => ({ label: t.title, value: t.title }))
    }
    return [{ label: '默认', value: '' }]
  }, [tabs])

  const loadExplore = async () => {
    if (!sourceId) return
    const thisId = ++requestIdRef.current
    setLoading(true)
    try {
      const res = (await api.getExplore(sourceId, tab || undefined, page, pageSize)) as ExploreResponse
      // Ignore stale responses caused by race conditions
      if (thisId !== requestIdRef.current) return
      if (res.tabs?.length) setTabs(res.tabs)
      if (res.tab && res.tab !== tab) setTab(res.tab)
      const list = res.books ?? res.items ?? []
      setBooks(list)
      setTotal(res.total ?? list.length)
      setHasMore(Boolean(res.hasMore))
    } catch (e: unknown) {
      // Ignore stale responses caused by race conditions
      if (thisId !== requestIdRef.current) return
      message.error(e instanceof Error ? e.message : '加载失败')
      setBooks([])
      setTabs([])
      setTotal(0)
      setHasMore(false)
    } finally {
      if (thisId === requestIdRef.current) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    setTab('')
    setTabs([])
    setBooks([])
    setPage(1)
    setTotal(0)
    setHasMore(false)
  }, [sourceId])

  useEffect(() => {
    setPage(1)
  }, [tab])

  useEffect(() => {
    if (sourceId) loadExplore()
  }, [sourceId, tab, page, pageSize]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleAdd = async (book: ExploreBook) => {
    try {
      await addShelfBook({
        bookKey: book.bookKey,
        name: book.name,
        author: book.author || '',
        coverUrl: book.coverUrl,
        sourceName: book.sourceName,
      } as ShelfBook)
      message.success(`已添加《${book.name}》`)
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '添加失败')
    }
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <h1 className="text-2xl font-bold mb-4">书海发现</h1>
      <div className="flex flex-wrap gap-3 mb-4">
        <Select
          className="min-w-[200px]"
          placeholder="选择书源"
          value={sourceId}
          onChange={setSourceId}
          options={(sources || []).map((s) => ({ label: s.name, value: s.id }))}
        />
        <Select
          className="min-w-[160px]"
          placeholder="分类 Tab"
          value={tab || undefined}
          onChange={(v) => setTab(v || '')}
          options={tabOptions}
        />
        <Text type="secondary" className="self-center">
          共 {total} 本{hasMore ? '，支持翻页' : ''}
        </Text>
      </div>

      {loading ? (
        <div className="py-16 text-center"><Spin tip="解析书海中..." /></div>
      ) : !books || books.length === 0 ? (
        <Empty description="暂无书单，请确认书源含 exploreUrl / exploreRule" />
      ) : (
        <div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {books.map((book) => {
              const inShelf = isBookInShelf(book.bookKey)
              return (
                <Card
                  key={book.bookKey}
                  hoverable
                  size="small"
                  className="overflow-hidden"
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
                          <ReadOutlined className="text-4xl text-gray-300" />
                        </div>
                      )}
                      {book.sourceName && (
                        <Tag className="absolute top-2 left-2" color="blue" style={{ fontSize: 10 }}>
                          {book.sourceName}
                        </Tag>
                      )}
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
                      <div>
                        <Text type="secondary" className="text-xs">
                          {book.author || '未知作者'}
                        </Text>
                        {book.intro && (
                          <Paragraph
                            type="secondary"
                            className="text-xs mt-1 mb-0 line-clamp-2"
                            style={{ fontSize: 11 }}
                          >
                            {book.intro}
                          </Paragraph>
                        )}
                      </div>
                    }
                  />
                  <div className="flex gap-1 mt-2">
                    {inShelf ? (
                      <Button size="small" disabled className="flex-1">已在书架</Button>
                    ) : (
                      <Button
                        size="small"
                        type="primary"
                        icon={<PlusOutlined />}
                        className="flex-1"
                        onClick={() => handleAdd(book)}
                      >
                        加入书架
                      </Button>
                    )}
                    <Button
                      size="small"
                      icon={<ReadOutlined />}
                      onClick={() => {
                        if (!book.bookKey?.trim()) {
                          message.warning('书籍标识无效，无法打开阅读器')
                          return
                        }
                        navigate(`/reader/${encodeURIComponent(book.bookKey)}`)
                      }}
                    />
                  </div>
                </Card>
              )
            })}
          </div>
          <div className="mt-6 flex justify-center">
            <Pagination
              current={page}
              pageSize={pageSize}
              total={total}
              showSizeChanger
              pageSizeOptions={['30', '60', '100', '200', '500']}
              onChange={(nextPage, nextPageSize) => {
                setPage(nextPage)
                if (nextPageSize !== pageSize) {
                  setPageSize(nextPageSize)
                }
              }}
              onShowSizeChange={(_, nextPageSize) => {
                setPage(1)
                setPageSize(nextPageSize)
              }}
            />
          </div>
        </div>
      )}
    </div>
  )
}
