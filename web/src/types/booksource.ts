/** 与后端 booksource.BookSource / Legado JSON 1:1 */
export type ParseMode = 'default' | 'xpath' | 'jsonpath' | 'css' | 'regex' | 'js'

export interface BookSourceDTO {
  id?: number
  name: string
  baseUrl: string
  searchUrl: string
  bookInfoUrl?: string
  tocUrl?: string
  contentUrl?: string
  searchRule: string
  bookInfoRule?: string
  tocRule?: string
  contentRule?: string
  searchMode?: ParseMode
  bookInfoMode?: ParseMode
  tocMode?: ParseMode
  contentMode?: ParseMode
  userAgent?: string
  headers?: string
  cookie?: string
  timeout?: number
  enabled: boolean
  group?: string
  order?: number
  exploreUrl?: string
  exploreRule?: string
  exploreMode?: ParseMode
  loginUrl?: string
  createdAt?: string
  updatedAt?: string
}

export const EMPTY_BOOK_SOURCE: BookSourceDTO = {
  name: '',
  baseUrl: '',
  searchUrl: '',
  searchRule: '',
  searchMode: 'default',
  bookInfoMode: 'default',
  tocMode: 'default',
  contentMode: 'default',
  enabled: true,
  group: '',
  headers: '{}',
  timeout: 15,
}

/** 解析书源 headers 字段（Legado 导入数据格式不统一） */
export function parseBookSourceHeaders(headers?: string): Record<string, string> {
  const raw = (headers || '').trim()
  if (!raw) return {}

  try {
    const parsed = JSON.parse(raw) as unknown
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, string>
    }
  } catch {
    // 继续尝试其他格式
  }

  if (/^Mozilla\//i.test(raw)) {
    return { 'User-Agent': raw }
  }

  if (raw.startsWith('{') && raw.includes("'")) {
    try {
      const parsed = JSON.parse(raw.replace(/'/g, '"')) as unknown
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, string>
      }
    } catch {
      // ignore
    }
  }

  return {}
}

/** 将 API 原始对象规范为 DTO（兼容旧字段 mode / headers object） */
export function normalizeBookSource(raw: Record<string, unknown>): BookSourceDTO {
  let headers = '{}'
  if (typeof raw.headers === 'string') {
    headers = raw.headers || '{}'
  } else if (raw.headers && typeof raw.headers === 'object') {
    headers = JSON.stringify(raw.headers)
  }

  return {
    id: typeof raw.id === 'number' ? raw.id : Number(raw.id) || undefined,
    name: String(raw.name ?? ''),
    baseUrl: String(raw.baseUrl ?? ''),
    searchUrl: String(raw.searchUrl ?? ''),
    bookInfoUrl: String(raw.bookInfoUrl ?? ''),
    tocUrl: String(raw.tocUrl ?? ''),
    contentUrl: String(raw.contentUrl ?? ''),
    searchRule: String(raw.searchRule ?? ''),
    bookInfoRule: String(raw.bookInfoRule ?? ''),
    tocRule: String(raw.tocRule ?? ''),
    contentRule: String(raw.contentRule ?? ''),
    searchMode: (raw.searchMode ?? raw.mode ?? 'default') as ParseMode,
    bookInfoMode: (raw.bookInfoMode ?? 'default') as ParseMode,
    tocMode: (raw.tocMode ?? 'default') as ParseMode,
    contentMode: (raw.contentMode ?? 'default') as ParseMode,
    userAgent: String(raw.userAgent ?? ''),
    headers,
    cookie: String(raw.cookie ?? ''),
    timeout: typeof raw.timeout === 'number' ? raw.timeout : 15,
    enabled: raw.enabled !== false,
    group: String(raw.group ?? ''),
    order: typeof raw.order === 'number' ? raw.order : 0,
    exploreUrl: String(raw.exploreUrl ?? ''),
    exploreRule: String(raw.exploreRule ?? ''),
    exploreMode: (raw.exploreMode ?? 'default') as ParseMode,
    loginUrl: String(raw.loginUrl ?? ''),
    createdAt: raw.createdAt as string | undefined,
    updatedAt: raw.updatedAt as string | undefined,
  }
}

export interface BookAlternateCandidate {
  sourceId: number
  sourceName: string
  bookKey: string
  name?: string
  author?: string
  matchScore?: number
  chapterScore?: number
  chapterIndex?: number
  coverUrl?: string
}

export interface SyncBundle {
  version: number
  exportedAt: string
  shelf: unknown[]
  bookSources: BookSourceDTO[]
  replaceRules: unknown[]
}

export interface SourceStat {
  sourceId: number
  lastSuccess: number
  lastError: number
  errorCount: number
  successCount: number
}
