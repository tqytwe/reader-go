import axios from 'axios'

const API_BASE_URL = '/api'

/** 统一解包后端 { code, message, data } 响应 */
export function unwrap<T = unknown>(res: unknown): T {
  if (res && typeof res === 'object' && 'code' in (res as object)) {
    const body = res as { code: number; message?: string; data?: T }
    if (body.code !== 0) {
      throw new Error(body.message || 'request failed')
    }
    return body.data as T
  }
  return res as T
}

const client = axios.create({
  baseURL: API_BASE_URL,
  timeout: 120000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
client.interceptors.request.use(
  (config) => {
    // 可在此添加认证 token
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor
client.interceptors.response.use(
  (response) => unwrap(response.data),
  (error) => {
    // 统一错误处理
    const msg = error?.response?.data?.message || error.message
    console.error('API Error:', msg)
    return Promise.reject(new Error(msg))
  }
)

export default client

// API 方法占位
export const api = {
  // Book sources
  getBookSources: () => client.get('/bookSources'),
  createBookSource: (data: any) => client.post('/bookSources', data),
  updateBookSource: (id: string, data: any) => client.put(`/bookSources/${id}`, data),
  deleteBookSource: (id: string) => client.delete(`/bookSources/${id}`),
  importBookSources: (data: any) => client.post('/bookSources/import', data),
  importBookSourceCollection: (url: string, enableOnlyNonJS = true) =>
    client.post('/bookSources/import/collection', { url, enableOnlyNonJS }),
  batchSetBookSourceEnabled: (payload: {
    target?: 'js' | 'nonJs' | 'all'
    enabled?: boolean
    enableOnlyNonJS?: boolean
  }) => client.post('/bookSources/batch/enable', payload),

  // RSS
  importRssSourceCollection: (url: string) => client.post('/rss/import/collection', { url }),

  // Search
  searchBooks: (q: string) =>
    client.get('/search', { params: { q } }),

  // Book
  getBookInfo: (bookKey: string) => client.get('/book/info', { params: { bookKey } }),
  getBookToc: (bookKey: string) => client.get('/book/toc', { params: { bookKey } }),
  getBookContent: (bookKey: string, chapter: string) =>
    client.get('/book/content', { params: { bookKey, chapter } }),

  // Shelf
  getShelf: () => client.get('/shelf'),
  addToShelf: (data: any) => client.post('/shelf', data),
  updateShelfBook: (id: string, data: any) => client.put(`/shelf/${id}`, data),
  updateShelfProgress: (id: number, data: { currentChapter: string; chapterIndex: number }) =>
    client.put(`/shelf/${id}/progress`, data),
  removeFromShelf: (id: number) => client.delete(`/shelf/${id}`),
  removeFromShelfByKey: (bookKey: string) =>
    client.delete('/shelf/0', { params: { bookKey } }),

  // Replace rules
  getReplaceRules: () => client.get('/replaceRules'),
  createReplaceRule: (data: any) => client.post('/replaceRules', data),
  updateReplaceRule: (id: string, data: any) => client.put(`/replaceRules/${id}`, data),
  deleteReplaceRule: (id: string) => client.delete(`/replaceRules/${id}`),

  // Local books
  uploadLocalBook: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return client.post('/localBooks', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  listLocalBooks: () => client.get('/localBooks'),
  getLocalBook: (id: string) => client.get(`/localBooks/${id}`),
  getLocalBookContent: (id: string, chapter?: number) =>
    client.get(`/localBooks/${id}/content`, {
      params: chapter !== undefined ? { chapter } : {},
    }),
  deleteLocalBook: (id: string) => client.delete(`/localBooks/${id}`),

  // Explore
  getExplore: (sourceId: number, tab?: string, page = 1, pageSize = 30) =>
    client.get('/explore', { params: { sourceId, tab, page, pageSize } }),

  // Alternates / sync / stats
  getBookAlternates: (bookKey: string) =>
    client.get('/book/alternates', { params: { bookKey } }),
  syncExport: () => client.get('/sync/export'),
  syncImport: (bundle: unknown) => client.post('/sync/import', bundle),
  getBookSourceStats: () => client.get('/bookSources/stats'),
}

export interface LocalBook {
  id: string
  name: string
  author: string
  format: string
  fileSize: number
  chapters: { title: string; index: number }[]
  coverUrl?: string
  createdAt: string
}

// RSS Types
export interface RssFeed {
  id: number
  title: string
  link: string
  description: string
  feedUrl: string
  siteUrl: string
  iconUrl: string
  feedType: number
  group: string
  enabled: boolean
  lastFetch: string
}

export interface RssItem {
  id: number
  feedId: number
  guid: string
  title: string
  link: string
  description: string
  content: string
  author: string
  publishedAt: string
  isRead: boolean
  isStarred: boolean
}

export interface RssItemsResponse {
  items: RssItem[]
  page: number
  pageSize: number
  total: number
  hasMore: boolean
}

export interface RssPreviewResult {
  feed: RssFeed
  resolvedUrl: string
  total: number
  items: RssItem[]
}

// RSS API（拦截器已解包，断言为 Promise<T>）
export const rssApi = {
  getRssFeeds: (): Promise<RssFeed[]> => client.get('/rss/feeds') as Promise<RssFeed[]>,
  addRssFeed: (url: string): Promise<RssFeed> => client.post('/rss/feeds', { feedUrl: url }) as Promise<RssFeed>,
  deleteRssFeed: (id: number): Promise<void> => client.delete(`/rss/feeds/${id}`) as Promise<void>,
  previewRssFeed: (id: number, payload?: { parseRules?: string; feedUrl?: string; siteUrl?: string; limit?: number }): Promise<RssPreviewResult> =>
    client.post(`/rss/feeds/${id}/preview`, payload ?? {}) as Promise<RssPreviewResult>,
  fetchRssFeed: (id: number): Promise<{ newItems?: number }> =>
    client.post(`/rss/feeds/${id}/fetch`) as Promise<{ newItems?: number }>,
  getRssItems: (feedId: number, page = 1, pageSize = 20): Promise<RssItemsResponse> =>
    client.get(`/rss/feeds/${feedId}/items`, { params: { page, pageSize } }) as Promise<RssItemsResponse>,
  markRssItemRead: (itemId: number): Promise<void> => client.put(`/rss/items/${itemId}/read`) as Promise<void>,
  toggleRssItemStar: (itemId: number): Promise<void> => client.put(`/rss/items/${itemId}/star`) as Promise<void>,
  importRssSourceCollection: (url: string) => client.post('/rss/import/collection', { url }),
}
