import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  LeftOutlined,
  RightOutlined,
  FontSizeOutlined,
  MinusOutlined,
  PlusOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  ThunderboltOutlined,
  MoonOutlined,
  SunOutlined,
  VerticalAlignTopOutlined,
  VerticalAlignBottomOutlined,
  SwapOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import { Button, Tooltip, Drawer, List, Spin } from 'antd'
import { THEME_CLASSES, useReader } from './useReader'
import type { Theme } from './types'

// ── Theme icon mapping ──────────────────────────────────────────────
const THEME_ICONS: Record<Theme, React.ReactNode> = {
  light: <SunOutlined />,
  dark: <MoonOutlined />,
  eye: <ThunderboltOutlined />,
}

// ── Helper: render plain text content (avoid XSS) ────────────────────
function ContentRenderer({ content, fontSettings }: { content: string; fontSettings: any }) {
  const htmlContent = useMemo(() => {
    // 如果内容已经包含 HTML 标签（如 <p>, <br>, <div>），直接使用
    if (/<(?:p|br|div|h[1-6])[\s>]/i.test(content)) {
      return content
        .replace(/<script[\s\S]*?<\/script>/gi, '')
        .replace(/<style[\s\S]*?<\/style>/gi, '')
        .replace(/javascript:/gi, '')
        .replace(/on\w+=/gi, '')
    }
    // 纯文本：将每行转换为 <p> 标签
    return content
      .split('\n')
      .map(line => {
        const trimmed = line.trim()
        if (!trimmed) return ''
        const escaped = trimmed
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
        return `<p>${escaped}</p>`
      })
      .join('\n')
  }, [content])

  return (
    <div
      className="reader-content prose max-w-none"
      style={{
        fontSize: `${fontSettings.size}px`,
        fontFamily: fontSettings.fontFamily,
        lineHeight: fontSettings.lineHeight,
        letterSpacing: `${fontSettings.letterSpacing}px`,
      }}
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  )
}

function ComicRenderer({ images, chapterName }: { images: string[]; chapterName: string }) {
  return (
    <div className="space-y-4">
      {images.map((src, index) => (
        <img
          key={`${chapterName}-${index}-${src}`}
          src={src}
          alt={`${chapterName} 第 ${index + 1} 页`}
          loading="lazy"
          className="block w-full h-auto rounded-lg shadow-sm bg-white"
        />
      ))}
    </div>
  )
}

// ── Progress bar ─────────────────────────────────────────────────────
function ProgressBar({
  percent,
  currentChapter,
}: {
  percent: number
  currentChapter: string | undefined
}) {
  return (
    <div className="flex items-center gap-3 px-4 py-2 border-t" style={{ borderColor: 'var(--reader-border)' }}>
      <span className="text-xs whitespace-nowrap" style={{ color: 'var(--reader-muted)' }}>
        {currentChapter || '加载中...'}
      </span>
      <div className="flex-1 h-1.5 rounded-full overflow-hidden" style={{ backgroundColor: 'var(--reader-hover)' }}>
        <div
          className="h-full bg-[#1677ff] transition-all duration-300"
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-xs whitespace-nowrap" style={{ color: 'var(--reader-muted)' }}>
        {percent}%
      </span>
    </div>
  )
}

// ── Sidebar TOC (mobile only; desktop uses aside) ───────────────────
function TOCSidebar({
  open,
  toc,
  currentChapterIndex,
  onSelect,
  onClose,
}: {
  open: boolean
  toc: any[]
  currentChapterIndex: number
  onSelect: (index: number) => void
  onClose: () => void
}) {
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(max-width: 767px)').matches,
  )

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)')
    const update = () => setIsMobile(mq.matches)
    update()
    mq.addEventListener('change', update)
    return () => mq.removeEventListener('change', update)
  }, [])

  // Flatten nested TOC for display
  const flatItems = useRef<any[]>([])

  useEffect(() => {
    const flatten = (items: any[], depth = 0): any[] =>
      items.reduce((acc, item) => {
        acc.push({ ...item, depth })
        if (item.subItems?.length) acc.push(...flatten(item.subItems, depth + 1))
        return acc
      }, [] as any[])

    flatItems.current = flatten(toc)
  }, [toc])

  if (!isMobile) return null

  return (
    <Drawer
      title="目录"
      placement="left"
      width={280}
      open={open}
      onClose={onClose}
      destroyOnClose
      styles={{
        body: { padding: 0 },
        header: { padding: '12px 16px' },
      }}
      className="toc-drawer"
    >
      <List
        dataSource={flatItems.current}
        renderItem={(item, i) => (
          <List.Item
            className={`cursor-pointer transition-colors ${
              item.name === toc[currentChapterIndex]?.name
                ? 'bg-[rgba(22,119,255,0.08)] dark:bg-[rgba(22,119,255,0.12)]'
                : ''
            }`}
            style={{
              paddingLeft: `${16 + item.depth * 16}px`,
              minHeight: 44,
              '--tw-bg-opacity': '1',
            } as React.CSSProperties}
            onMouseEnter={(e) => {
              if (item.name !== toc[currentChapterIndex]?.name) {
                e.currentTarget.style.backgroundColor = 'var(--reader-hover)'
              }
            }}
            onMouseLeave={(e) => {
              if (item.name !== toc[currentChapterIndex]?.name) {
                e.currentTarget.style.backgroundColor = ''
              }
            }}
            onClick={() => {
              onSelect(i)
              onClose()
            }}
          >
            <span className="text-sm truncate block py-3 pr-3">{item.name}</span>
          </List.Item>
        )}
      />
    </Drawer>
  )
}

// ── Font settings panel ──────────────────────────────────────────────
function FontSettingsPanel({
  fontSettings,
  onIncrease,
  onDecrease,
  onReset,
}: {
  fontSettings: any
  onIncrease: () => void
  onDecrease: () => void
  onReset: () => void
}) {
  return (
    <div className="flex items-center gap-2">
      <Tooltip title="减小字号">
        <Button
          size="small"
          icon={<MinusOutlined />}
          onClick={onDecrease}
          disabled={fontSettings.size <= 12}
        />
      </Tooltip>
      <span className="text-sm min-w-[32px] text-center">{fontSettings.size}</span>
      <Tooltip title="增大字号">
        <Button
          size="small"
          icon={<PlusOutlined />}
          onClick={onIncrease}
          disabled={fontSettings.size >= 32}
        />
      </Tooltip>
      <Button size="small" icon={<FontSizeOutlined />} onClick={onReset} type="text">
        重置
      </Button>
    </div>
  )
}

// ── Main Reader Component ────────────────────────────────────────────
export default function Reader() {
  const navigate = useNavigate()
  const contentRef = useRef<HTMLDivElement>(null)

  const {
    bookInfo,
    toc,
    content,
    loading,
    error,

    sidebarOpen,
    theme,
    scrollMode,
    showControls,

    currentChapterIndex,
    currentChapter,
    hasNext,
    hasPrev,

    goToChapter,
    goPrev,
    goNext,
    firstChapter,
    lastChapter,

    setSidebarOpen,
    setTheme,
    setScrollMode,
    setShowControls,
    showControlsTemporarily,

    increaseFontSize,
    decreaseFontSize,
    resetFontSize,

    fontSettings,
    progressPercent,
  } = useReader()

  // Keyboard navigation
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement).tagName)) return

      switch (e.key) {
        case 'ArrowLeft':
          e.preventDefault()
          goPrev()
          break
        case 'ArrowRight':
          e.preventDefault()
          goNext()
          break
        case 'ArrowUp':
        case 'ArrowDown':
          showControlsTemporarily()
          break
        case 'Home':
          e.preventDefault()
          firstChapter()
          break
        case 'End':
          e.preventDefault()
          lastChapter()
          break
        case 'Escape':
          setSidebarOpen(false)
          break
      }
    }

    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [goPrev, goNext, firstChapter, lastChapter, showControlsTemporarily, setSidebarOpen])

  // Touch swipe for mobile
  const touchStartX = useRef(0)
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
    showControlsTemporarily()
  }
  const handleTouchEnd = (e: React.TouchEvent) => {
    const diff = touchStartX.current - e.changedTouches[0].clientX
    if (Math.abs(diff) > 60) {
      if (diff > 0) goNext()
      else goPrev()
    }
  }

  // Theme change effect on document
  useEffect(() => {
    document.body.classList.remove('theme-light', 'theme-dark', 'theme-eye')
    document.body.classList.add(`theme-${theme}`)
    return () => {
      document.body.classList.remove(`theme-${theme}`)
    }
  }, [theme])

  // Render loading state
  if (loading) {
    return (
      <div className={`h-full flex items-center justify-center ${THEME_CLASSES[theme]}`}>
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  // Render error state
  if (error) {
    return (
      <div className={`h-full flex items-center justify-center ${THEME_CLASSES[theme]}`}>
        <div className="text-center">
          <p className="text-red-500 mb-4">{error}</p>
          <Button onClick={() => navigate(-1)}>返回</Button>
        </div>
      </div>
    )
  }

  return (
    <div
      className={`h-full flex flex-col transition-colors duration-300 ${THEME_CLASSES[theme]}`}
      onMouseMove={showControlsTemporarily}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      {/* ── Top bar ─────────────────────────────────────────────────── */}
      <header
        className={`flex items-center justify-between px-4 py-2 border-b transition-opacity duration-300 ${
          showControls ? 'opacity-100' : 'opacity-0 hover:opacity-100'
        }`}
        style={{ borderColor: 'var(--reader-border)' }}
      >
        <div className="flex items-center gap-2">
          <Button
            size="small"
            icon={sidebarOpen ? <MenuFoldOutlined /> : <MenuUnfoldOutlined />}
            onClick={() => setSidebarOpen(!sidebarOpen)}
          />
          <Button size="small" icon={<LeftOutlined />} onClick={() => navigate(-1)} />
          <span className="text-sm font-medium truncate max-w-[200px]">
            {bookInfo?.name || '阅读中'}
          </span>
        </div>

        <div className="flex items-center gap-1">
          {/* Theme switcher */}
          <Tooltip title="主题">
            <div className="flex items-center border rounded-md overflow-hidden">
              {(
                [
                  ['light', '白天'],
                  ['dark', '夜间'],
                  ['eye', '护眼'],
                ] as [Theme, string][]
              ).map(([t, label]) => (
                <button
                  key={t}
                  className={`p-1.5 transition-colors ${
                    theme === t ? 'bg-[#1677ff] text-white' : 'hover:bg-[rgba(0,0,0,0.05)]'
                  }`}
                  onClick={() => setTheme(t)}
                  title={label}
                >
                  {THEME_ICONS[t]}
                </button>
              ))}
            </div>
          </Tooltip>

          {/* Font size */}
          <FontSettingsPanel
            fontSettings={fontSettings}
            onIncrease={increaseFontSize}
            onDecrease={decreaseFontSize}
            onReset={resetFontSize}
          />

          {/* Scroll mode toggle */}
          <Tooltip title={scrollMode === 'scroll' ? '翻页模式' : '滚动模式'}>
            <Button
              size="small"
              icon={<SwapOutlined />}
              onClick={() => setScrollMode(scrollMode === 'scroll' ? 'page' : 'scroll')}
              type="text"
            />
          </Tooltip>

          {/* Controls visibility */}
          <Tooltip title={showControls ? '隐藏控件' : '显示控件'}>
            <Button
              size="small"
              icon={showControls ? <EyeInvisibleOutlined /> : <EyeOutlined />}
              onClick={() => setShowControls(!showControls)}
              type="text"
            />
          </Tooltip>
        </div>
      </header>

      {/* ── Main content area ───────────────────────────────────────── */}
      <div className="flex-1 flex overflow-hidden">
        {/* TOC Sidebar (desktop) */}
        {sidebarOpen && (
          <aside
            className="w-64 border-r overflow-y-auto hidden md:block"
            style={{ borderColor: 'var(--reader-border)' }}
          >
            <div
              className="p-3 border-b sticky top-0 bg-inherit"
              style={{ borderColor: 'var(--reader-border)' }}
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">目录</span>
                <Button
                  size="small"
                  icon={<CloseOutlined />}
                  onClick={() => setSidebarOpen(false)}
                  type="text"
                />
              </div>
            </div>
            <nav className="py-2">
              {toc.map((item, i) => (
                <button
                  key={item.url || i}
                  className={`w-full text-left px-4 py-2 text-sm truncate transition-colors ${
                    item.name === currentChapter?.name
                      ? 'bg-[rgba(22,119,255,0.08)] text-[#1677ff] font-medium'
                      : ''
                  }`}
                  onMouseEnter={(e) => {
                    if (item.name !== currentChapter?.name) {
                      e.currentTarget.style.backgroundColor = 'var(--reader-hover)'
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (item.name !== currentChapter?.name) {
                      e.currentTarget.style.backgroundColor = ''
                    }
                  }}
                  onClick={() => goToChapter(i)}
                >
                  {item.name}
                </button>
              ))}
            </nav>
          </aside>
        )}

        {/* Mobile TOC drawer */}
        <TOCSidebar
          open={sidebarOpen}
          toc={toc}
          currentChapterIndex={currentChapterIndex}
          onSelect={goToChapter}
          onClose={() => setSidebarOpen(false)}
        />

        {/* Content area */}
        <main
          ref={contentRef}
          className={`flex-1 overflow-y-auto ${scrollMode === 'page' ? 'scroll-smooth' : ''}`}
        >
          <div
            className={`max-w-[800px] mx-auto px-4 sm:px-6 md:px-8 py-8 ${
              scrollMode === 'page' ? 'min-h-[calc(100vh-140px)]' : ''
            }`}
          >
            {/* Book info header */}
            {bookInfo && (
              <div
                className="mb-8 pb-6 border-b"
                style={{ borderColor: 'var(--reader-border)' }}
              >
                <h1 className="text-2xl font-bold mb-1">{bookInfo.name}</h1>
                <p className="text-sm" style={{ color: 'var(--reader-muted)' }}>
                  {bookInfo.author}
                </p>
              </div>
            )}

            {/* Chapter content */}
            {content ? (
              <article>
                <h2 className="text-xl font-semibold mb-6 text-center">
                  {content.chapterName}
                </h2>
                {content.readerMode === 'comic' && (content.images?.length ?? 0) > 0 ? (
                  <ComicRenderer images={content.images ?? []} chapterName={content.chapterName} />
                ) : (
                  <ContentRenderer content={content.content} fontSettings={fontSettings} />
                )}
              </article>
            ) : (
              <div
                className="text-center py-12"
                style={{ color: 'var(--reader-muted)' }}
              >
                <p>章节内容加载中...</p>
              </div>
            )}
          </div>
        </main>
      </div>

      {/* ── Bottom navigation bar ───────────────────────────────────── */}
      <footer
        className={`border-t transition-opacity duration-300 ${
          showControls ? 'opacity-100' : 'opacity-0 hover:opacity-100'
        }`}
        style={{ borderColor: 'var(--reader-border)' }}
      >
        {/* Navigation buttons */}
        <div className="flex items-center justify-center gap-2 px-4 py-2">
          <Tooltip title="上一章 (←)">
            <Button size="small" icon={<LeftOutlined />} onClick={goPrev} disabled={!hasPrev}>
              上一章
            </Button>
          </Tooltip>

          <Tooltip title="下一章 (→)">
            <Button size="small" icon={<RightOutlined />} onClick={goNext} disabled={!hasNext}>
              下一章
            </Button>
          </Tooltip>

          <div
            className="w-px h-5 mx-2"
            style={{ backgroundColor: 'var(--reader-muted)', opacity: 0.2 }}
          />

          <Tooltip title="第一章">
            <Button
              size="small"
              icon={<VerticalAlignTopOutlined />}
              onClick={firstChapter}
              type="text"
            />
          </Tooltip>

          <Tooltip title="最新章">
            <Button
              size="small"
              icon={<VerticalAlignBottomOutlined />}
              onClick={lastChapter}
              type="text"
            />
          </Tooltip>
        </div>

        {/* Progress bar */}
        <ProgressBar percent={progressPercent} currentChapter={currentChapter?.name} />
      </footer>

      {/* ── Mobile: tap center area to show controls ────────────────── */}
      {!showControls && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none md:hidden">
          <div className="pointer-events-auto">
            <Button size="large" icon={<EyeOutlined />} onClick={showControlsTemporarily}>
              显示菜单
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
