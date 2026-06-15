// Reader types

export interface BookInfo {
  bookKey: string
  name: string
  author: string
  cover?: string
  intro?: string
  lastChapter?: string
  updateTime?: string
}

export interface TOCItem {
  name: string
  url: string
  subItems?: TOCItem[]
}

export interface ChapterContent {
  chapterName: string
  content: string
  images?: string[]
  readerMode?: 'text' | 'comic'
}

export type Theme = 'light' | 'dark' | 'eye'

export type ScrollMode = 'scroll' | 'page'

export interface ReadingProgress {
  bookKey: string
  currentChapter: string
  currentChapterIndex: number
  scrollPosition: number
  timestamp: number
}

export interface FontSettings {
  size: number
  fontFamily: string
  lineHeight: number
  letterSpacing: number
}
