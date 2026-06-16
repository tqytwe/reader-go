import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { api } from '../api/client'

// ===== Book Source Types =====
export type ParseMode = 'default' | 'xpath' | 'jsonpath' | 'css' | 'regex' | 'js'

export interface BookSource {
  id: string
  name: string
  baseUrl: string
  searchUrl: string
  searchRule: string
  mode: ParseMode
  enabled: boolean
  group: string
  headers: Record<string, string>
  createdAt?: string
  updatedAt?: string
}

export interface BookSourceFormValues {
  name: string
  baseUrl: string
  searchUrl: string
  searchRule: string
  mode: ParseMode
  enabled: boolean
  group: string
  headersJson: string // JSON string for headers
}

// ===== Shelf Book Types =====
export interface ShelfBook {
  // Backend fields (match Go struct JSON tags)
  id?: number
  bookKey: string
  name: string
  author: string
  coverUrl?: string
  summary?: string
  sourceId?: number
  sourceName?: string
  currentChapter?: string
  lastReadAt?: string
  readCount?: number
  note?: string
  order?: number
  createdAt?: string
  updatedAt?: string

  // Frontend-only computed/derived fields
  progress?: number // 阅读进度 0-100 (computed)
  totalChapters?: number
  group?: string // 自定义分组
  bookUrl?: string // legacy fallback
}

// ===== Search History =====
export interface SearchHistoryItem {
  keyword: string
  timestamp: number
}

// ===== Shelf Store =====
export interface ShelfState {
  // 书架书籍列表
  books: ShelfBook[]

  // Actions
  setBooks: (books: ShelfBook[]) => void
  addBook: (book: ShelfBook) => void
  updateBook: (bookKey: string, data: Partial<ShelfBook>) => void
  removeBook: (bookKey: string) => Promise<void>
  clearBooks: () => void

  // 检查书籍是否在书架
  isBookInShelf: (bookKey: string) => boolean

  // 获取书籍
  getBook: (bookKey: string) => ShelfBook | undefined

  // 批量操作
  addBooks: (books: ShelfBook[]) => void
  removeBooks: (bookKeys: string[]) => void
}

export const useShelfStore = create<ShelfState>()(
  persist(
    (set, get) => ({
      books: [],

      setBooks: (books) => set({ books: normalizeShelfBooks(books) }),

      addBook: (book) =>
        set((state) => {
          const existingIndex = state.books.findIndex((b) => b.bookKey === book.bookKey)
          if (existingIndex >= 0) {
            // 更新已存在的书籍
            const updated = [...state.books]
            updated[existingIndex] = { ...updated[existingIndex], ...book, createdAt: book.createdAt ?? updated[existingIndex].createdAt }
            return { books: updated }
          }
          return { books: [...state.books, { ...book, createdAt: book.createdAt ?? new Date().toISOString() }] }
        }),

      updateBook: (bookKey, data) =>
        set((state) => ({
          books: state.books.map((b) =>
            b.bookKey === bookKey ? { ...b, ...data } : b
          ),
        })),

      removeBook: async (bookKey) => {
        const book = get().books.find((b) => b.bookKey === bookKey)
        try {
          if (book?.id) {
            await api.removeFromShelf(book.id)
          } else {
            await api.removeFromShelfByKey(bookKey)
          }
        } catch (e) {
          console.error('remove from shelf API failed:', e)
          throw e
        }
        set((state) => ({
          books: state.books.filter((b) => b.bookKey !== bookKey),
        }))
      },

      clearBooks: () => set({ books: [] }),

      isBookInShelf: (bookKey) =>
        get().books.some((b) => b.bookKey === bookKey),

      getBook: (bookKey) =>
        get().books.find((b) => b.bookKey === bookKey),

      addBooks: (newBooks) =>
        set((state) => {
          const existingKeys = new Set(state.books.map((b) => b.bookKey))
          const toAdd = newBooks.filter((b) => !existingKeys.has(b.bookKey))
          return {
            books: [
              ...state.books,
              ...toAdd.map((b) => ({ ...b, createdAt: b.createdAt ?? new Date().toISOString() })),
            ],
          }
        }),

      removeBooks: (bookKeys) =>
        set((state) => ({
          books: state.books.filter((b) => !bookKeys.includes(b.bookKey)),
        })),
    }),
    {
      name: 'shelf-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ books: state.books }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<ShelfState> | undefined
        return {
          ...current,
          ...saved,
          books: normalizeShelfBooks(saved?.books),
        }
      },
    }
  )
)

function normalizeShelfBooks(books: unknown): ShelfBook[] {
  if (Array.isArray(books)) {
    return books
  }
  if (books && typeof books === 'object' && Array.isArray((books as { books?: ShelfBook[] }).books)) {
    return (books as { books: ShelfBook[] }).books
  }
  return []
}

// ===== Search Store =====
export interface SearchState {
  // 搜索历史
  searchHistory: SearchHistoryItem[]
  // 最大历史记录数
  maxHistory: number

  // Actions
  setSearchHistory: (history: SearchHistoryItem[]) => void
  addSearchHistory: (keyword: string) => void
  removeSearchHistory: (keyword: string) => void
  clearSearchHistory: () => void
}

export const useSearchStore = create<SearchState>()(
  persist(
    (set) => ({
      searchHistory: [],
      maxHistory: 20,

      setSearchHistory: (history) => set({ searchHistory: history }),

      addSearchHistory: (keyword) =>
        set((state) => {
          const normalized = keyword.trim()
          if (!normalized) return state

          // 移除旧的相同关键词
          const filtered = state.searchHistory.filter(
            (item) => item.keyword !== normalized
          )

          // 添加到开头
          const newItem: SearchHistoryItem = {
            keyword: normalized,
            timestamp: Date.now(),
          }

          // 限制数量
          const history = [newItem, ...filtered].slice(0, state.maxHistory)
          return { searchHistory: history }
        }),

      removeSearchHistory: (keyword) =>
        set((state) => ({
          searchHistory: state.searchHistory.filter(
            (item) => item.keyword !== keyword
          ),
        })),

      clearSearchHistory: () => set({ searchHistory: [] }),
    }),
    {
      name: 'search-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ searchHistory: state.searchHistory }),
    }
  )
)

// ===== Combined Store (for backward compatibility) =====
interface AppState {
  // 书架
  books: ShelfBook[]
  // 书源
  bookSources: BookSource[]
  // 替换规则
  replaceRules: any[]
  // 书源 loading 状态
  bookSourcesLoading: boolean
  // 书源搜索关键词
  bookSourceSearchKeyword: string
  // 书源分组筛选
  bookSourceGroupFilter: string

  // Actions
  setBooks: (books: ShelfBook[]) => void
  setBookSources: (sources: BookSource[]) => void
  setBookSourcesLoading: (loading: boolean) => void
  setBookSourceSearchKeyword: (keyword: string) => void
  setBookSourceGroupFilter: (group: string) => void
  addBookSource: (source: BookSource) => void
  updateBookSource: (id: string, data: Partial<BookSource>) => void
  removeBookSource: (id: string) => void
  toggleBookSourceEnabled: (id: string) => void
  setReplaceRules: (rules: any[]) => void

  // Shelf actions (API + local store)
  addShelfBook: (book: ShelfBook) => Promise<void>
  removeShelfBook: (bookKey: string) => Promise<void>
  refreshShelf: () => Promise<void>
}

export const useStore = create<AppState>((set) => ({
  books: [],
  bookSources: [],
  replaceRules: [],
  bookSourcesLoading: false,
  bookSourceSearchKeyword: '',
  bookSourceGroupFilter: '',

  setBooks: (books) => set({ books }),
  setBookSources: (sources) => set({ bookSources: sources }),
  setBookSourcesLoading: (loading) => set({ bookSourcesLoading: loading }),
  setBookSourceSearchKeyword: (keyword) => set({ bookSourceSearchKeyword: keyword }),
  setBookSourceGroupFilter: (group) => set({ bookSourceGroupFilter: group }),
  addBookSource: (source) =>
    set((state) => ({ bookSources: [...state.bookSources, source] })),
  updateBookSource: (id, data) =>
    set((state) => ({
      bookSources: state.bookSources.map((s) =>
        s.id === id ? { ...s, ...data, updatedAt: new Date().toISOString() } : s
      ),
    })),
  removeBookSource: (id) =>
    set((state) => ({
      bookSources: state.bookSources.filter((s) => s.id !== id),
    })),
  toggleBookSourceEnabled: (id) =>
    set((state) => ({
      bookSources: state.bookSources.map((s) =>
        s.id === id ? { ...s, enabled: !s.enabled, updatedAt: new Date().toISOString() } : s
      ),
    })),
  setReplaceRules: (rules) => set({ replaceRules: rules }),

  addShelfBook: async (book) => {
    const parts = book.bookKey.split('::')
    const sourceId = book.sourceId ?? (parts[0] ? parseInt(parts[0], 10) : 0)
    const saved = (await api.addToShelf({
      bookKey: book.bookKey,
      name: book.name,
      author: book.author,
      coverUrl: book.coverUrl,
      summary: book.summary,
      sourceId: Number.isFinite(sourceId) ? sourceId : 0,
      sourceName: book.sourceName,
    })) as unknown as ShelfBook
    useShelfStore.getState().addBook({ ...book, ...saved })
  },
  removeShelfBook: (bookKey) => useShelfStore.getState().removeBook(bookKey),
  refreshShelf: async () => {
    const data = (await api.getShelf()) as { books?: ShelfBook[] }
    useShelfStore.getState().setBooks(data?.books ?? [])
  },
}))

// ===== Lock Store (密码锁屏) =====
const LOCK_STORAGE_KEY = 'reader_go_lock'
const LOCK_SESSION_KEY = 'reader_go_unlocked'

async function hashPassword(password: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(password + '_reader_go_salt')
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

export interface LockState {
  /** 是否已设置密码 */
  isPasswordSet: boolean
  /** 密码哈希 */
  passwordHash: string | null

  /** 设置密码 */
  setPassword: (password: string) => Promise<void>
  /** 验证密码 */
  verifyPassword: (password: string) => Promise<boolean>
  /** 清除密码（关闭锁屏） */
  clearPassword: () => void
  /** 是否已解锁（session 级别） */
  isUnlocked: () => boolean
  /** 标记已解锁 */
  unlock: () => void
  /** 锁定 */
  lock: () => void
}

export const useLockStore = create<LockState>()(
  (set, get) => ({
    isPasswordSet: false,
    passwordHash: null,

    setPassword: async (password) => {
      const hash = await hashPassword(password)
      localStorage.setItem(LOCK_STORAGE_KEY, JSON.stringify({ hash }))
      set({ isPasswordSet: true, passwordHash: hash })
    },

    verifyPassword: async (password) => {
      const hash = await hashPassword(password)
      return hash === get().passwordHash
    },

    clearPassword: () => {
      localStorage.removeItem(LOCK_STORAGE_KEY)
      sessionStorage.removeItem(LOCK_SESSION_KEY)
      set({ isPasswordSet: false, passwordHash: null })
    },

    isUnlocked: () => {
      return sessionStorage.getItem(LOCK_SESSION_KEY) === '1'
    },

    unlock: () => {
      sessionStorage.setItem(LOCK_SESSION_KEY, '1')
    },

    lock: () => {
      sessionStorage.removeItem(LOCK_SESSION_KEY)
    },
  })
)

// 初始化锁屏状态
function initLockState() {
  try {
    const stored = localStorage.getItem(LOCK_STORAGE_KEY)
    if (stored) {
      const { hash } = JSON.parse(stored)
      if (hash) {
        useLockStore.setState({ isPasswordSet: true, passwordHash: hash })
      }
    }
  } catch {
    // ignore
  }
}
initLockState()
