import { useState, useEffect, useMemo } from 'react'
import { rssApi, RssFeed, RssItem, RssItemsResponse, RssPreviewResult } from '../api/client'
import { message, Modal, Input } from 'antd'
import { SearchOutlined } from '@ant-design/icons'

export default function Rss() {
  const [feeds, setFeeds] = useState<RssFeed[]>([])
  const [items, setItems] = useState<RssItem[]>([])
  const [preview, setPreview] = useState<RssPreviewResult | null>(null)
  const [selectedFeed, setSelectedFeed] = useState<RssFeed | null>(null)
  const [groupFilter, setGroupFilter] = useState<string>('all')
  const [newFeedUrl, setNewFeedUrl] = useState('')
  const [collectionUrl, setCollectionUrl] = useState('')
  const [importingCollection, setImportingCollection] = useState(false)
  const [loading, setLoading] = useState(false)
  const [addingFeed, setAddingFeed] = useState(false)
  const [fetchingFeed, setFetchingFeed] = useState<number | null>(null)
  const [previewingFeed, setPreviewingFeed] = useState<number | null>(null)
  const [panelMode, setPanelMode] = useState<'items' | 'preview'>('items')
  const [itemsPage, setItemsPage] = useState(1)
  const [itemsPageSize, setItemsPageSize] = useState(20)
  const [itemsTotal, setItemsTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [feedSearch, setFeedSearch] = useState('')
  const [selectedFeedIds, setSelectedFeedIds] = useState<number[]>([])

  // Load feeds
  useEffect(() => {
    loadFeeds()
  }, [])

  const loadFeeds = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await rssApi.getRssFeeds()
      setFeeds(data)
    } catch (err) {
      setError('加载订阅源失败')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  // Get unique groups
  const groups = useMemo(() => {
    if (!feeds || feeds.length === 0) return ['all']
    const groupSet = new Set(feeds.map(f => f.group || 'Default'))
    return ['all', ...Array.from(groupSet)]
  }, [feeds])

  // Filtered feeds
  const filteredFeeds = useMemo(() => {
    if (!feeds) return []
    let result = feeds
    if (groupFilter !== 'all') {
      result = result.filter(f => (f.group || 'Default') === groupFilter)
    }
    if (feedSearch.trim()) {
      const q = feedSearch.toLowerCase()
      result = result.filter(f =>
        (f.title || '').toLowerCase().includes(q) ||
        (f.feedUrl || '').toLowerCase().includes(q)
      )
    }
    return result
  }, [feeds, groupFilter, feedSearch])

  // 链接导入订阅源合集
  const handleImportCollection = async () => {
    if (!collectionUrl.trim()) {
      message.warning('请输入订阅源合集链接')
      return
    }
    setImportingCollection(true)
    setError(null)
    try {
      const result = await rssApi.importRssSourceCollection(collectionUrl.trim()) as { imported?: number; total?: number }
      message.success(`成功导入 ${result?.imported ?? 0} / ${result?.total ?? 0} 个订阅源`)
      setCollectionUrl('')
      await loadFeeds()
    } catch (err: any) {
      setError(err?.message || '导入订阅源合集失败')
      message.error(err?.message || '导入失败')
    } finally {
      setImportingCollection(false)
    }
  }

  // Add feed
  const handleAddFeed = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newFeedUrl.trim()) return

    setAddingFeed(true)
    setError(null)
    try {
      await rssApi.addRssFeed(newFeedUrl.trim())
      setNewFeedUrl('')
      await loadFeeds()
    } catch (err) {
      setError('添加订阅源失败')
      message.error('添加订阅源失败')
      console.error(err)
    } finally {
      setAddingFeed(false)
    }
  }

  // Delete feed
  const handleDeleteFeed = async (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个订阅源吗？',
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await rssApi.deleteRssFeed(id)
          if (selectedFeed?.id === id) {
            setSelectedFeed(null)
            setItems([])
            setPreview(null)
            setItemsTotal(0)
          }
          await loadFeeds()
          message.success('已删除')
        } catch (err) {
          setError('删除订阅源失败')
          message.error('删除订阅源失败')
          console.error(err)
        }
      },
    })
  }

  // Batch delete feeds
  const handleBatchDeleteFeeds = () => {
    if (selectedFeedIds.length === 0) {
      message.warning('请先选择要删除的订阅源')
      return
    }
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedFeedIds.length} 个订阅源吗？`,
      okText: '确认删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await Promise.all(selectedFeedIds.map(id => rssApi.deleteRssFeed(id)))
          if (selectedFeedIds.includes(selectedFeed?.id ?? -1)) {
            setSelectedFeed(null)
            setItems([])
            setPreview(null)
            setItemsTotal(0)
          }
          setSelectedFeedIds([])
          message.success(`已删除 ${selectedFeedIds.length} 个订阅源`)
          await loadFeeds()
        } catch (err) {
          setError('批量删除失败')
          message.error('批量删除失败')
          console.error(err)
        }
      },
    })
  }

  // Toggle feed selection
  const toggleFeedSelection = (feedId: number, e: React.MouseEvent) => {
    e.stopPropagation()
    setSelectedFeedIds(prev =>
      prev.includes(feedId)
        ? prev.filter(id => id !== feedId)
        : [...prev, feedId]
    )
  }

  // Refresh feed
  const handleRefreshFeed = async (feed: RssFeed) => {
    setFetchingFeed(feed.id)
    try {
      const result = await rssApi.fetchRssFeed(feed.id)
      if (selectedFeed?.id === feed.id) {
        await loadItems(feed.id, 1, itemsPageSize)
        setPanelMode('items')
      }
      const count = result?.newItems ?? 0
      message.success(count > 0 ? `刷新成功，新增 ${count} 篇文章` : '刷新完成，但当前没有新增条目')
    } catch (err) {
      const msg = err instanceof Error ? err.message : '刷新订阅源失败，请检查订阅地址或网络'
      setError(msg)
      message.error('刷新订阅源失败')
      console.error(err)
    } finally {
      setFetchingFeed(null)
    }
  }

  // Load items for selected feed
  const loadItems = async (feedId: number, page = itemsPage, pageSize = itemsPageSize) => {
    try {
      const data = await rssApi.getRssItems(feedId, page, pageSize)
      const result = data as RssItemsResponse
      setItems(result.items)
      setItemsPage(result.page)
      setItemsPageSize(result.pageSize)
      setItemsTotal(result.total)
    } catch (err) {
      setError('加载 RSS 条目失败')
      console.error(err)
    }
  }

  // Select feed
  const handleSelectFeed = async (feed: RssFeed) => {
    setSelectedFeed(feed)
    setPreview(null)
    setPanelMode('items')
    setItemsPage(1)
    await loadItems(feed.id, 1, itemsPageSize)
  }

  const handlePreviewFeed = async (feed: RssFeed) => {
    setPreviewingFeed(feed.id)
    setSelectedFeed(feed)
    setError(null)
    try {
      const result = await rssApi.previewRssFeed(feed.id, { limit: 10 })
      setPreview(result)
      setPanelMode('preview')
      message.success(`预览成功，解析到 ${result.total} 条`)
    } catch (err) {
      const msg = err instanceof Error ? err.message : '预览订阅源失败，请检查规则或网络'
      setError(msg)
      message.error('预览订阅源失败')
      console.error(err)
    } finally {
      setPreviewingFeed(null)
    }
  }

  // Mark read
  const handleMarkRead = async (itemId: number) => {
    try {
      await rssApi.markRssItemRead(itemId)
      setItems(items.map(item =>
        item.id === itemId ? { ...item, isRead: true } : item
      ))
    } catch (err) {
      console.error(err)
    }
  }

  // Toggle star
  const handleToggleStar = async (itemId: number) => {
    try {
      await rssApi.toggleRssItemStar(itemId)
      setItems(items.map(item =>
        item.id === itemId ? { ...item, isStarred: !item.isStarred } : item
      ))
    } catch (err) {
      console.error(err)
    }
  }

  // Open link
  const handleOpenLink = (link: string) => {
    window.open(link, '_blank', 'noopener,noreferrer')
  }

  // Format date
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const days = Math.floor(diff / (1000 * 60 * 60 * 24))

    if (days === 0) return '今天'
    if (days === 1) return '昨天'
    if (days < 7) return `${days} 天前`
    if (days < 30) return `${Math.floor(days / 7)} 周前`
    return date.toLocaleDateString('zh-CN')
  }

  return (
    <div className="flex h-[calc(100vh-4rem)]">
      {/* Feed List Sidebar */}
      <div className="w-80 border-r bg-gray-50 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b bg-white">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold">RSS 订阅</h2>
            {selectedFeedIds.length > 0 && (
              <button
                onClick={handleBatchDeleteFeeds}
                className="px-3 py-1 bg-red-500 text-white rounded text-sm hover:bg-red-600"
              >
                批量删除 ({selectedFeedIds.length})
              </button>
            )}
          </div>

          {/* 搜索订阅源 */}
          <div className="mb-3">
            <Input
              placeholder="搜索订阅源..."
              prefix={<SearchOutlined />}
              allowClear
              size="small"
              value={feedSearch}
              onChange={(e) => setFeedSearch(e.target.value)}
            />
          </div>

          {/* 链接导入合集 */}
          <div className="flex gap-2 mb-3">
            <input
              type="url"
              value={collectionUrl}
              onChange={(e) => setCollectionUrl(e.target.value)}
              placeholder="订阅源合集链接..."
              className="flex-1 px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="button"
              disabled={importingCollection}
              onClick={handleImportCollection}
              className="px-3 py-2 bg-green-500 text-white rounded-lg text-sm hover:bg-green-600 disabled:opacity-50 whitespace-nowrap"
            >
              {importingCollection ? '...' : '合集导入'}
            </button>
          </div>

          {/* Add Feed Form */}
          <form onSubmit={handleAddFeed} className="flex gap-2 mb-3">
            <input
              type="url"
              value={newFeedUrl}
              onChange={(e) => setNewFeedUrl(e.target.value)}
              placeholder="输入订阅源 URL..."
              className="flex-1 px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={addingFeed}
              className="px-3 py-2 bg-blue-500 text-white rounded-lg text-sm hover:bg-blue-600 disabled:opacity-50"
            >
              {addingFeed ? '...' : '添加'}
            </button>
          </form>

          {/* Group Filter */}
          <select
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
            className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {groups.map(group => (
              <option key={group} value={group}>
                {group === 'all' ? '全部分组' : group}
              </option>
            ))}
          </select>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mx-4 mt-3 p-2 bg-red-100 text-red-700 rounded-lg text-sm">
            {error}
          </div>
        )}

        {/* Feed List */}
        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="p-4 text-center text-gray-500">加载中...</div>
          ) : filteredFeeds.length === 0 ? (
            <div className="p-4 text-center text-gray-500">
              {feedSearch ? '未找到匹配的订阅源' : '暂无订阅源，请添加'}
            </div>
          ) : (
            <div className="divide-y">
              {filteredFeeds.map(feed => (
                <div
                  key={feed.id}
                  className={`p-3 hover:bg-gray-100 cursor-pointer transition ${
                    selectedFeed?.id === feed.id ? 'bg-blue-50 border-l-4 border-blue-500' : ''
                  } ${selectedFeedIds.includes(feed.id) ? 'bg-blue-50' : ''}`}
                  onClick={() => handleSelectFeed(feed)}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-2 flex-1 min-w-0">
                      <input
                        type="checkbox"
                        checked={selectedFeedIds.includes(feed.id)}
                        onClick={(e) => toggleFeedSelection(feed.id, e)}
                        className="mt-1 cursor-pointer"
                      />
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-sm truncate">{feed.title || 'Untitled'}</h3>
                        <p className="text-xs text-gray-500 truncate mt-1">{feed.feedUrl}</p>
                        {feed.group && (
                          <span className="inline-block mt-1 px-2 py-0.5 bg-gray-200 rounded text-xs">
                            {feed.group}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1 ml-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handlePreviewFeed(feed)
                        }}
                        disabled={previewingFeed === feed.id}
                        className="p-1.5 hover:bg-blue-100 rounded disabled:opacity-50 text-blue-600"
                        title="Preview"
                      >
                        <svg className={`w-4 h-4 ${previewingFeed === feed.id ? 'animate-pulse' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                        </svg>
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleRefreshFeed(feed)
                        }}
                        disabled={fetchingFeed === feed.id}
                        className="p-1.5 hover:bg-gray-200 rounded disabled:opacity-50"
                        title="Refresh"
                      >
                        <svg className={`w-4 h-4 ${fetchingFeed === feed.id ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleDeleteFeed(feed.id)
                        }}
                        className="p-1.5 hover:bg-red-100 rounded text-red-600"
                        title="Delete"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Items Panel */}
      <div className="flex-1 flex flex-col bg-white">
        {selectedFeed ? (
          <>
            {/* Items Header */}
            <div className="p-4 border-b">
              <h2 className="text-xl font-semibold">{selectedFeed.title}</h2>
              <p className="text-sm text-gray-500 mt-1">{selectedFeed.description}</p>
              <div className="flex items-center gap-2 mt-3">
                <button
                  type="button"
                  onClick={() => setPanelMode('items')}
                  className={`px-3 py-1.5 rounded-lg text-sm ${panelMode === 'items' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-700'}`}
                >
                  已抓取条目 {itemsTotal > 0 ? `(${itemsTotal})` : ''}
                </button>
                <button
                  type="button"
                  onClick={() => setPanelMode('preview')}
                  disabled={!preview}
                  className={`px-3 py-1.5 rounded-lg text-sm ${panelMode === 'preview' ? 'bg-emerald-600 text-white' : 'bg-gray-100 text-gray-700 disabled:opacity-50'}`}
                >
                  规则预览
                </button>
              </div>
            </div>

            {/* Items List */}
            <div className="flex-1 overflow-y-auto">
              {panelMode === 'preview' ? (
                !preview ? (
                  <div className="p-8 text-center text-gray-500">
                    先点击左侧订阅源的预览按钮，再查看规则解析结果。
                  </div>
                ) : (
                  <div className="p-4">
                    <div className="mb-4 rounded-xl border bg-emerald-50 p-4">
                      <div className="text-sm text-emerald-900 font-medium">预览概览</div>
                      <div className="mt-2 text-sm text-emerald-800">解析总数：{preview.total}</div>
                      <div className="mt-1 text-xs text-emerald-700 break-all">resolvedUrl: {preview.resolvedUrl}</div>
                      <div className="mt-3 flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={fetchingFeed === selectedFeed.id}
                          onClick={() => handleRefreshFeed(selectedFeed)}
                          className="px-3 py-1.5 rounded-lg bg-emerald-600 text-white text-sm disabled:opacity-50"
                        >
                          抓取并写入条目
                        </button>
                        <button
                          type="button"
                          onClick={() => setPanelMode('items')}
                          className="px-3 py-1.5 rounded-lg bg-white text-emerald-700 border text-sm"
                        >
                          查看已抓取条目
                        </button>
                      </div>
                    </div>
                    <div className="divide-y rounded-xl border bg-white">
                      {(preview.items || []).map((item: RssItem, index: number) => (
                        <div key={`${item.guid}-${index}`} className="p-4">
                          <div className="font-medium text-gray-900">{item.title || 'Untitled'}</div>
                          <div className="mt-1 text-xs text-gray-500 break-all">{item.link}</div>
                          <div className="mt-2 text-sm text-gray-600">{item.description || item.content || 'No description'}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )
              ) : !items || items.length === 0 ? (
                <div className="p-8 text-center text-gray-500">
                  <div className="text-base text-gray-700">这个订阅源当前没有已抓取条目。</div>
                  <div className="mt-2 text-sm">
                    可以先点左侧“刷新”抓取，或先看“规则预览”确认当前源是否真的能解析。
                  </div>
                  {error && (
                    <div className="mt-3 text-xs text-red-600 break-all">
                      最近错误：{error}
                    </div>
                  )}
                </div>
              ) : (
                <div>
                  <div className="px-4 py-3 border-b bg-gray-50 text-sm text-gray-600 flex items-center justify-between">
                    <span>第 {itemsPage} 页，共 {itemsTotal} 条</span>
                    <div className="flex items-center gap-2">
                      <select
                        value={itemsPageSize}
                        onChange={async (e) => {
                          const nextPageSize = Number(e.target.value)
                          setItemsPage(1)
                          setItemsPageSize(nextPageSize)
                          if (selectedFeed) {
                            await loadItems(selectedFeed.id, 1, nextPageSize)
                          }
                        }}
                        className="px-2 py-1 border rounded text-sm bg-white"
                      >
                        {[10, 20, 50, 100].map((size) => (
                          <option key={size} value={size}>{size} / 页</option>
                        ))}
                      </select>
                    </div>
                  </div>
                  <div className="divide-y">
                    {items.map(item => (
                      <div
                        key={item.id}
                        className={`p-4 hover:bg-gray-50 transition ${
                          !item.isRead ? 'bg-blue-50/50' : ''
                        }`}
                      >
                        <div className="flex items-start gap-3">
                          <button
                            onClick={() => handleToggleStar(item.id)}
                            className={`mt-1 ${item.isStarred ? 'text-yellow-500' : 'text-gray-300 hover:text-yellow-500'}`}
                          >
                            <svg className="w-5 h-5" fill={item.isStarred ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                            </svg>
                          </button>

                          <div className="flex-1 min-w-0">
                            <div className="flex items-start justify-between gap-2">
                              <h3 className={`font-medium ${!item.isRead ? 'text-blue-900' : 'text-gray-900'}`}>
                                {item.title}
                              </h3>
                              <span className="text-xs text-gray-400 whitespace-nowrap">
                                {formatDate(item.publishedAt)}
                              </span>
                            </div>

                            {item.author && (
                              <p className="text-xs text-gray-500 mt-1">By {item.author}</p>
                            )}

                            <p className="text-sm text-gray-600 mt-2 line-clamp-2">
                              {item.description || item.content?.slice(0, 200)}
                            </p>

                            <div className="flex items-center gap-3 mt-3">
                              {!item.isRead && (
                                <button
                                  onClick={() => handleMarkRead(item.id)}
                                  className="text-xs text-blue-600 hover:text-blue-700"
                                >
                                  标记已读
                                </button>
                              )}
                              <button
                                onClick={() => handleOpenLink(item.link)}
                                className="text-xs text-gray-600 hover:text-gray-900 flex items-center gap-1"
                              >
                                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                                </svg>
                                打开链接
                              </button>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className="flex items-center justify-center gap-2 p-4 border-t bg-white">
                    <button
                      type="button"
                      disabled={itemsPage <= 1}
                      onClick={async () => {
                        if (selectedFeed && itemsPage > 1) {
                          await loadItems(selectedFeed.id, itemsPage - 1, itemsPageSize)
                        }
                      }}
                      className="px-3 py-1.5 rounded border text-sm disabled:opacity-50"
                    >
                      上一页
                    </button>
                    <span className="text-sm text-gray-600">
                      {itemsPage} / {Math.max(1, Math.ceil(itemsTotal / itemsPageSize))}
                    </span>
                    <button
                      type="button"
                      disabled={itemsPage * itemsPageSize >= itemsTotal}
                      onClick={async () => {
                        if (selectedFeed && itemsPage * itemsPageSize < itemsTotal) {
                          await loadItems(selectedFeed.id, itemsPage + 1, itemsPageSize)
                        }
                      }}
                      className="px-3 py-1.5 rounded border text-sm disabled:opacity-50"
                    >
                      下一页
                    </button>
                  </div>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-400">
            <div className="text-center">
              <svg className="w-16 h-16 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
              </svg>
              <p>选择左侧订阅源查看文章</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
