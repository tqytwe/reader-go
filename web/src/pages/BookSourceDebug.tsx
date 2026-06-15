import { useState, useEffect, useRef, useMemo } from 'react'
import { Select, Input, Button, Card, Tag, message, Spin, Empty } from 'antd'
import { BugOutlined, PlayCircleOutlined, ClearOutlined, InfoCircleOutlined } from '@ant-design/icons'
import { useStore, type BookSource } from '@/store/useStore'

type DebugType = 'search' | 'info' | 'toc' | 'content'

interface DebugResult {
  step: string
  status: 'start' | 'done' | 'error'
  message: string
  data: any
  timestamp: number
}

const DEBUG_TYPES: { label: string; value: DebugType; description: string }[] = [
  { label: '搜索 (search)', value: 'search', description: '搜索书籍' },
  { label: '详情 (info)', value: 'info', description: '获取书籍详情' },
  { label: '目录 (toc)', value: 'toc', description: '获取书籍目录' },
  { label: '正文 (content)', value: 'content', description: '获取章节正文' },
]

const STEP_LABELS: Record<string, string> = {
  search: '搜索',
  info: '详情',
  toc: '目录',
  content: '正文',
}

const STATUS_COLORS: Record<string, string> = {
  start: 'blue',
  done: 'green',
  error: 'red',
}

const STATUS_LABELS: Record<string, string> = {
  start: '开始',
  done: '完成',
  error: '错误',
}

function JsonViewer({ data, defaultExpanded = false }: { data: any; defaultExpanded?: boolean }) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  if (data === null || data === undefined) {
    return <span className="text-gray-400">null</span>
  }

  const jsonStr = JSON.stringify(data, null, 2)

  if (jsonStr.length < 200 && !expanded) {
    return <code className="text-sm text-gray-700 dark:text-gray-300">{jsonStr}</code>
  }

  return (
    <div className="text-sm">
      <Button
        type="link"
        size="small"
        onClick={() => setExpanded(!expanded)}
        className="p-0 h-auto text-xs"
      >
        {expanded ? '折叠' : '展开'}
      </Button>
      {expanded ? (
        <pre className="mt-2 p-3 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-auto max-h-96 whitespace-pre-wrap break-all">
          {jsonStr}
        </pre>
      ) : (
        <code className="text-gray-500 text-xs">{jsonStr.slice(0, 200)}{jsonStr.length > 200 ? '...' : ''}</code>
      )}
    </div>
  )
}

function ResultItem({ result }: { result: DebugResult }) {
  return (
    <div className="border-l-4 border-gray-200 dark:border-gray-600 pl-4 py-3 mb-3">
      <div className="flex items-center gap-2 flex-wrap">
        <Tag color={STATUS_COLORS[result.status]}>{STATUS_LABELS[result.status]}</Tag>
        <span className="font-medium text-gray-700 dark:text-gray-300">
          {STEP_LABELS[result.step] || result.step}
        </span>
        <span className="text-gray-500 text-sm">{result.message}</span>
      </div>
      {result.data !== null && result.data !== undefined && (
        <div className="mt-2">
          <JsonViewer data={result.data} />
        </div>
      )}
    </div>
  )
}

export default function BookSourceDebug() {
  const { bookSources, bookSourcesLoading } = useStore()

  const [selectedSourceId, setSelectedSourceId] = useState<string | null>(null)
  const [debugType, setDebugType] = useState<DebugType>('search')
  const [query, setQuery] = useState('')
  const [bookUrl, setBookUrl] = useState('')
  const [chapterUrl, setChapterUrl] = useState('')
  const [results, setResults] = useState<DebugResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const eventSourceRef = useRef<EventSource | null>(null)
  const resultsEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new results come in
  useEffect(() => {
    resultsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [results])

  // Cleanup EventSource on unmount
  useEffect(() => {
    return () => {
      eventSourceRef.current?.close()
    }
  }, [])

  // Get book source groups
  const groups = useMemo(() => {
    const groupMap = new Map<string, BookSource[]>()
    bookSources.forEach((source) => {
      const group = source.group || '未分组'
      if (!groupMap.has(group)) {
        groupMap.set(group, [])
      }
      groupMap.get(group)!.push(source)
    })
    return groupMap
  }, [bookSources])

  // Get selected source
  const selectedSource = useMemo(() => {
    return bookSources.find((s) => s.id === selectedSourceId)
  }, [bookSources, selectedSourceId])

  const handleStartDebug = () => {
    if (!selectedSourceId) {
      message.warning('请选择书源')
      return
    }

    if (debugType === 'search' && !query.trim()) {
      message.warning('请输入搜索关键词')
      return
    }

    if (['info', 'toc', 'content'].includes(debugType) && !bookUrl.trim()) {
      message.warning('请输入书籍 URL')
      return
    }

    if (debugType === 'content' && !chapterUrl.trim()) {
      message.warning('请输入章节 URL')
      return
    }

    // Close existing connection
    eventSourceRef.current?.close()

    setLoading(true)
    setError(null)
    setResults([])

    // Build URL
    const params = new URLSearchParams({
      sourceId: selectedSourceId,
      type: debugType,
    })

    if (debugType === 'search') {
      params.set('query', query.trim())
    } else {
      params.set('bookUrl', bookUrl.trim())
      if (debugType === 'content') {
        params.set('chapterUrl', chapterUrl.trim())
      }
    }

    const url = `/api/bookSources/debug?${params.toString()}`
    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as DebugResult
        data.timestamp = Date.now()
        setResults((prev) => [...prev, data])
      } catch (e) {
        console.error('Failed to parse SSE data:', e)
      }
    }

    eventSource.onerror = () => {
      setError('连接失败，请检查网络或重试')
      setLoading(false)
      eventSource.close()
    }

    eventSource.addEventListener('done', () => {
      setLoading(false)
      eventSource.close()
    })
  }

  const handleClear = () => {
    setResults([])
    setError(null)
  }

  const handleStop = () => {
    eventSourceRef.current?.close()
    setLoading(false)
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <BugOutlined />
          书源调试
        </h1>
      </div>

      <Card className="mb-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* 书源选择 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              书源
            </label>
            <Select
              placeholder="选择书源"
              value={selectedSourceId}
              onChange={setSelectedSourceId}
              className="w-full"
              loading={bookSourcesLoading}
              showSearch
              filterOption={(input, option) =>
                typeof option?.label === 'string' &&
                option.label.toLowerCase().includes(input.toLowerCase())
              }
            >
              {Array.from(groups.entries()).map(([group, sources]) => (
                <Select.OptGroup key={group} label={group}>
                  {sources.map((source) => (
                    <Select.Option
                      key={source.id}
                      value={source.id}
                      label={source.name}
                    >
                      <div className="flex items-center gap-2">
                        <span>{source.name}</span>
                        <span className="text-xs text-gray-400">{source.baseUrl}</span>
                      </div>
                    </Select.Option>
                  ))}
                </Select.OptGroup>
              ))}
            </Select>
          </div>

          {/* 调试类型 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              调试类型
            </label>
            <Select
              value={debugType}
              onChange={(value) => setDebugType(value)}
              className="w-full"
              options={DEBUG_TYPES.map((t) => ({
                label: t.label,
                value: t.value,
              }))}
            />
          </div>

          {/* search 参数 */}
          {debugType === 'search' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                搜索关键词
              </label>
              <Input
                placeholder="输入搜索关键词"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onPressEnter={handleStartDebug}
              />
            </div>
          )}

          {/* info/toc/content 参数 */}
          {debugType !== 'search' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                书籍 URL
              </label>
              <Input
                placeholder="输入书籍 URL"
                value={bookUrl}
                onChange={(e) => setBookUrl(e.target.value)}
                onPressEnter={handleStartDebug}
              />
            </div>
          )}

          {/* content 额外参数 */}
          {debugType === 'content' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                章节 URL
              </label>
              <Input
                placeholder="输入章节 URL"
                value={chapterUrl}
                onChange={(e) => setChapterUrl(e.target.value)}
                onPressEnter={handleStartDebug}
              />
            </div>
          )}
        </div>

        {/* 操作按钮 */}
        <div className="flex gap-2 mt-4">
          {loading ? (
            <Button
              danger
              onClick={handleStop}
            >
              停止
            </Button>
          ) : (
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleStartDebug}
              disabled={!selectedSourceId}
            >
              开始调试
            </Button>
          )}
          <Button
            icon={<ClearOutlined />}
            onClick={handleClear}
            disabled={results.length === 0 && !error}
          >
            清空
          </Button>
        </div>
      </Card>

      {/* 选中的书源信息 */}
      {selectedSource && (
        <Card
          size="small"
          className="mb-6 bg-gray-50 dark:bg-gray-800"
        >
          <div className="flex items-center gap-4 text-sm">
            <InfoCircleOutlined />
            <span className="font-medium">{selectedSource.name}</span>
            <span className="text-gray-500">{selectedSource.baseUrl}</span>
            <Tag>{selectedSource.mode.toUpperCase()}</Tag>
          </div>
        </Card>
      )}

      {/* 错误信息 */}
      {error && (
        <Card className="mb-6 bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800">
          <div className="text-red-600 dark:text-red-400">
            {error}
          </div>
        </Card>
      )}

      {/* 结果展示 */}
      <Card
        title={
          <div className="flex items-center justify-between">
            <span>调试结果</span>
            {loading && <Spin size="small" />}
          </div>
        }
        className="min-h-[400px]"
      >
        {results.length === 0 && !loading && !error && (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="点击「开始调试」查看结果"
          />
        )}

        {results.length > 0 && (
          <div className="space-y-0">
            {results.map((result, index) => (
              <ResultItem key={`${result.timestamp}-${index}`} result={result} />
            ))}
            <div ref={resultsEndRef} />
          </div>
        )}
      </Card>
    </div>
  )
}