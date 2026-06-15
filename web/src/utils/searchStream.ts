export interface StreamSearchResult {
  bookKey?: string
  bookId?: string
  sourceId?: string
  name?: string
  bookName?: string
  author?: string
  intro?: string
  summary?: string
  sourceName?: string
  source?: string
  bookUrl?: string
  coverUrl?: string
  cover?: string
}

export interface StreamSearchHandlers {
  onResult: (item: StreamSearchResult) => void
  onDone: (total: number) => void
  onError: (message: string) => void
}

/** 通过 SSE 流式搜索，逐条回调结果 */
export function searchBooksStream(q: string, handlers: StreamSearchHandlers): () => void {
  const url = `/api/search/stream?q=${encodeURIComponent(q)}`
  const es = new EventSource(url)
  let finished = false

  es.addEventListener('result', (ev) => {
    try {
      const item = JSON.parse(ev.data) as StreamSearchResult
      handlers.onResult(item)
    } catch {
      if (!finished) {
        finished = true
        handlers.onError('解析搜索结果失败')
        es.close()
      }
    }
  })

  es.addEventListener('done', (ev) => {
    if (finished) return
    finished = true
    try {
      const payload = JSON.parse(ev.data) as { total?: number }
      handlers.onDone(payload.total ?? 0)
    } catch {
      handlers.onDone(0)
    }
    es.close()
  })

  es.addEventListener('server-error', (ev) => {
    if (finished) return
    finished = true
    try {
      const msg = JSON.parse(ev.data) as string
      handlers.onError(msg || '搜索失败')
    } catch {
      handlers.onError('搜索失败')
    }
    es.close()
  })

  es.onerror = () => {
    if (finished || es.readyState === EventSource.CONNECTING) return
    finished = true
    handlers.onError('流式搜索连接中断')
    es.close()
  }

  return () => {
    finished = true
    es.close()
  }
}

export function mapStreamResult(item: StreamSearchResult) {
  const bookKey =
    item.bookKey ||
    (item.sourceId && item.bookUrl ? `${item.sourceId}::${item.bookUrl}` : '') ||
    item.bookId ||
    ''
  return {
    bookId: bookKey,
    bookName: item.bookName || item.name || '',
    author: item.author || '未知作者',
    intro: item.intro || item.summary || '',
    sourceName: item.sourceName || item.source || '未知来源',
    bookUrl: item.bookUrl || bookKey,
    coverUrl: item.coverUrl || item.cover,
  }
}
