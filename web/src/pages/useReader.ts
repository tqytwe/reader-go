import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { api, type LocalBook } from '../api/client'
import { useShelfStore } from '../store/useStore'
import type { BookInfo, TOCItem, ChapterContent, Theme, ScrollMode, ReadingProgress, FontSettings } from './types'

const PROGRESS_STORAGE_KEY = 'reader_progress'
const SETTINGS_STORAGE_KEY = 'reader_settings'
const LOCAL_BOOK_PREFIX = 'local-'

function isLocalBookKey(bookKey: string): boolean {
  return bookKey.startsWith(LOCAL_BOOK_PREFIX)
}

function localBookIdFromKey(bookKey: string): string {
  return bookKey.slice(LOCAL_BOOK_PREFIX.length)
}

// Default font settings
const DEFAULT_FONT_SETTINGS: FontSettings = {
  size: 18,
  fontFamily: 'system-ui, -apple-system, sans-serif',
  lineHeight: 1.8,
  letterSpacing: 0.5,
}

// Theme CSS class mapping (use CSS variables defined in index.css)
export const THEME_CLASSES: Record<Theme, string> = {
  light: 'theme-light',
  dark: 'theme-dark',
  eye: 'theme-eye',
}

// Theme labels
export const THEME_LABELS: Record<Theme, string> = {
  light: '白天',
  dark: '夜间',
  eye: '护眼',
}

interface BookInfoResponse {
  name: string
  author: string
  coverUrl?: string
  summary?: string
  bookKey?: string
}

interface TocResponse {
  chapters?: { name: string; url: string }[]
}

interface ContentResponse {
  content: string
  chapter?: string
  images?: string[]
  readerMode?: 'text' | 'comic'
}

export function useReader() {
  const { bookId } = useParams<{ bookId: string }>()
  const bookKey = bookId || ''
  // Data state
  const [bookInfo, setBookInfo] = useState<BookInfo | null>(null)
  const [toc, setToc] = useState<TOCItem[]>([])
  const [content, setContent] = useState<ChapterContent | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // UI state
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [currentChapterIndex, setCurrentChapterIndex] = useState(0)
  const [theme, setTheme] = useState<Theme>('light')
  const [scrollMode, setScrollMode] = useState<ScrollMode>('scroll')
  const [showControls, setShowControls] = useState(true)
  const [controlsTimer, setControlsTimer] = useState<ReturnType<typeof setTimeout> | null>(null)

  // Font settings
  const [fontSettings, setFontSettings] = useState<FontSettings>(DEFAULT_FONT_SETTINGS)

  const progressSyncRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Load settings from localStorage on mount（与全局 app_theme 同步）
  useEffect(() => {
    try {
      const savedSettings = localStorage.getItem(SETTINGS_STORAGE_KEY)
      if (savedSettings) {
        const parsed = JSON.parse(savedSettings)
        if (parsed.theme) setTheme(parsed.theme)
        if (parsed.scrollMode) setScrollMode(parsed.scrollMode)
        if (parsed.fontSettings) setFontSettings({ ...DEFAULT_FONT_SETTINGS, ...parsed.fontSettings })
      } else {
        const appTheme = localStorage.getItem('app_theme')
        if (appTheme === 'light' || appTheme === 'dark' || appTheme === 'eye') {
          setTheme(appTheme)
        }
      }
    } catch (e) {
      console.error('Failed to load reader settings:', e)
    }
  }, [])

  // Save settings to localStorage when changed
  useEffect(() => {
    try {
      localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({
        theme,
        scrollMode,
        fontSettings,
      }))
      localStorage.setItem('app_theme', theme)
      document.documentElement.classList.remove('app-theme-light', 'app-theme-dark', 'app-theme-eye')
      document.documentElement.classList.add(`app-theme-${theme}`)
      document.documentElement.setAttribute('data-theme', theme)
    } catch (e) {
      console.error('Failed to save reader settings:', e)
    }
  }, [theme, scrollMode, fontSettings])

  // Load progress from localStorage
  const loadProgress = useCallback((bk: string): ReadingProgress | null => {
    try {
      const raw = localStorage.getItem(PROGRESS_STORAGE_KEY)
      if (!raw) return null
      const all: Record<string, ReadingProgress> = JSON.parse(raw)
      return all[bk] || null
    } catch {
      return null
    }
  }, [])

  // Save progress to localStorage
  const saveProgress = useCallback((progress: ReadingProgress) => {
    try {
      const raw = localStorage.getItem(PROGRESS_STORAGE_KEY)
      const all: Record<string, ReadingProgress> = raw ? JSON.parse(raw) : {}
      all[progress.bookKey] = progress
      localStorage.setItem(PROGRESS_STORAGE_KEY, JSON.stringify(all))
    } catch (e) {
      console.error('Failed to save reading progress:', e)
    }
  }, [])

  // Sync progress to server when book is on shelf
  const syncProgressToServer = useCallback((chapterName: string, chapterIndex: number) => {
    const shelfBook = useShelfStore.getState().getBook(bookKey)
    if (!shelfBook?.id) return

    if (progressSyncRef.current) clearTimeout(progressSyncRef.current)
    progressSyncRef.current = setTimeout(() => {
      api.updateShelfProgress(shelfBook.id!, {
        currentChapter: chapterName,
        chapterIndex,
      }).catch((e) => console.error('Failed to sync shelf progress:', e))
    }, 800)
  }, [bookKey])

  // Fetch book data
  useEffect(() => {
    if (!bookKey) return

    let cancelled = false
    const fetchAll = async () => {
      setLoading(true)
      setError(null)
      try {
        if (isLocalBookKey(bookKey)) {
          const localId = localBookIdFromKey(bookKey)
          const bookRes = await api.getLocalBook(localId)
          if (cancelled) return

          const info = bookRes as unknown as LocalBook
          const tocData: TOCItem[] = (info.chapters ?? []).map((ch) => ({
            name: ch.title,
            url: String(ch.index),
          }))

          setBookInfo({
            bookKey,
            name: info.name,
            author: info.author,
            cover: info.coverUrl,
          })
          setToc(tocData)

          const progress = loadProgress(bookKey)
          let startIndex = 0
          if (progress && tocData.length > 0) {
            const idx = tocData.findIndex((item) => item.name === progress.currentChapter)
            startIndex = idx >= 0 ? idx : 0
          }
          setCurrentChapterIndex(startIndex)

          if (tocData[startIndex]) {
            const chapterIdx = Number(tocData[startIndex].url)
            const contentRes = await api.getLocalBookContent(localId, chapterIdx)
            if (!cancelled) {
              const body = contentRes as unknown as ContentResponse
              setContent({
                chapterName: tocData[startIndex].name,
                content: body.content ?? '',
                images: body.images ?? [],
                readerMode: body.readerMode ?? ((body.images?.length ?? 0) > 0 ? 'comic' : 'text'),
              })
            }
          } else if (tocData.length === 0) {
            const contentRes = await api.getLocalBookContent(localId)
            if (!cancelled) {
              const body = contentRes as unknown as ContentResponse
              setContent({
                chapterName: info.name,
                content: body.content ?? '',
                images: body.images ?? [],
                readerMode: body.readerMode ?? ((body.images?.length ?? 0) > 0 ? 'comic' : 'text'),
              })
            }
          }
          return
        }

        const [bookRes, tocRes] = await Promise.all([
          api.getBookInfo(bookKey),
          api.getBookToc(bookKey),
        ])

        if (cancelled) return

        const info = bookRes as unknown as BookInfoResponse
        const tocBody = tocRes as unknown as TocResponse
        const tocData: TOCItem[] = (tocBody.chapters ?? []).map((ch) => ({
          name: ch.name,
          url: ch.url,
        }))

        setBookInfo({
          bookKey,
          name: info.name,
          author: info.author,
          cover: info.coverUrl,
          intro: info.summary,
        })
        setToc(tocData)

        const progress = loadProgress(bookKey)
        let startIndex = 0

        if (progress && tocData.length > 0) {
          const idx = tocData.findIndex((item) => item.name === progress.currentChapter)
          startIndex = idx >= 0 ? idx : 0
        }

        setCurrentChapterIndex(startIndex)

        if (tocData[startIndex]) {
          const contentRes = await api.getBookContent(bookKey, tocData[startIndex].url)
          if (!cancelled) {
            const body = contentRes as unknown as ContentResponse
            setContent({
              chapterName: tocData[startIndex].name,
              content: body.content ?? '',
              images: body.images ?? [],
              readerMode: body.readerMode ?? ((body.images?.length ?? 0) > 0 ? 'comic' : 'text'),
            })
          }
        }
      } catch (err: unknown) {
        if (!cancelled) {
          const msg = err instanceof Error ? err.message : 'Failed to load book data'
          setError(msg)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    fetchAll()
    return () => { cancelled = true }
  }, [bookKey, loadProgress])

  // Save progress when chapter changes
  useEffect(() => {
    if (!bookInfo || toc.length === 0) return

    const currentChapter = toc[currentChapterIndex]
    if (!currentChapter) return

    saveProgress({
      bookKey,
      currentChapter: currentChapter.name,
      currentChapterIndex,
      scrollPosition: 0,
      timestamp: Date.now(),
    })
    syncProgressToServer(currentChapter.name, currentChapterIndex)
  }, [currentChapterIndex, bookInfo, toc, bookKey, saveProgress, syncProgressToServer])

  // Fetch chapter content when index changes
  useEffect(() => {
    if (!bookInfo || toc.length === 0) return

    const currentChapter = toc[currentChapterIndex]
    if (!currentChapter) return

    const fetchContent = async () => {
      try {
        if (isLocalBookKey(bookKey)) {
          const localId = localBookIdFromKey(bookKey)
          const chapterIdx = Number(currentChapter.url)
          const res = chapterIdx >= 0 && !Number.isNaN(chapterIdx)
            ? await api.getLocalBookContent(localId, chapterIdx)
            : await api.getLocalBookContent(localId)
          const body = res as unknown as ContentResponse
          setContent({
            chapterName: currentChapter.name,
            content: body.content ?? '',
            images: body.images ?? [],
            readerMode: body.readerMode ?? ((body.images?.length ?? 0) > 0 ? 'comic' : 'text'),
          })
          return
        }

        const res = await api.getBookContent(bookKey, currentChapter.url)
        const body = res as unknown as ContentResponse
        setContent({
          chapterName: currentChapter.name,
          content: body.content ?? '',
          images: body.images ?? [],
          readerMode: body.readerMode ?? ((body.images?.length ?? 0) > 0 ? 'comic' : 'text'),
        })
      } catch (err: unknown) {
        console.error('Failed to load chapter:', err)
      }
    }

    fetchContent()
  }, [currentChapterIndex, bookKey, bookInfo, toc])

  // Navigation
  const goToChapter = useCallback((index: number) => {
    if (index >= 0 && index < toc.length) {
      setCurrentChapterIndex(index)
    }
  }, [toc.length])

  const goPrev = useCallback(() => {
    if (currentChapterIndex > 0) {
      goToChapter(currentChapterIndex - 1)
    }
  }, [currentChapterIndex, goToChapter])

  const goNext = useCallback(() => {
    if (currentChapterIndex < toc.length - 1) {
      goToChapter(currentChapterIndex + 1)
    }
  }, [currentChapterIndex, toc.length, goToChapter])

  const firstChapter = useCallback(() => {
    goToChapter(0)
  }, [goToChapter])

  const lastChapter = useCallback(() => {
    goToChapter(toc.length - 1)
  }, [goToChapter, toc.length])

  // Control visibility with auto-hide
  const showControlsTemporarily = useCallback(() => {
    setShowControls(true)
    if (controlsTimer) clearTimeout(controlsTimer)
    const timer = setTimeout(() => setShowControls(false), 2500)
    setControlsTimer(timer)
  }, [controlsTimer])

  // Font size helpers
  const increaseFontSize = useCallback(() => {
    setFontSettings((s) => ({ ...s, size: Math.min(s.size + 2, 32) }))
  }, [])

  const decreaseFontSize = useCallback(() => {
    setFontSettings((s) => ({ ...s, size: Math.max(s.size - 2, 12) }))
  }, [])

  const resetFontSize = useCallback(() => {
    setFontSettings(DEFAULT_FONT_SETTINGS)
  }, [])

  // Progress percentage
  const progressPercent = useMemo(() => {
    if (toc.length === 0) return 0
    return Math.round(((currentChapterIndex + 1) / toc.length) * 100)
  }, [currentChapterIndex, toc.length])

  return {
    // Data
    bookInfo,
    toc,
    content,
    loading,
    error,

    // UI state
    sidebarOpen,
    theme,
    scrollMode,
    showControls,
    fontSettings,

    // Actions
    setSidebarOpen,
    setTheme,
    setScrollMode,
    setShowControls,
    showControlsTemporarily,
    increaseFontSize,
    decreaseFontSize,
    resetFontSize,

    // Navigation
    currentChapterIndex,
    goToChapter,
    goPrev,
    goNext,
    firstChapter,
    lastChapter,

    // Computed
    progressPercent,
    currentChapter: toc[currentChapterIndex],
    hasNext: currentChapterIndex < toc.length - 1,
    hasPrev: currentChapterIndex > 0,
  }
}
