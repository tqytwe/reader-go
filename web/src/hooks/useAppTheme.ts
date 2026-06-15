import { useCallback, useEffect, useState } from 'react'

export type AppTheme = 'light' | 'dark' | 'eye'

const STORAGE_KEY = 'app_theme'

export const APP_THEME_CLASSES: Record<AppTheme, string> = {
  light: 'app-theme-light',
  dark: 'app-theme-dark',
  eye: 'app-theme-eye',
}

export const APP_THEME_LABELS: Record<AppTheme, string> = {
  light: '白天',
  dark: '夜间',
  eye: '护眼',
}

const THEME_ORDER: AppTheme[] = ['light', 'dark', 'eye']

function readStoredTheme(): AppTheme {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'eye') return v
    const readerSettings = localStorage.getItem('reader_settings')
    if (readerSettings) {
      const parsed = JSON.parse(readerSettings)
      if (parsed?.theme === 'light' || parsed?.theme === 'dark' || parsed?.theme === 'eye') {
        return parsed.theme
      }
    }
  } catch {
    /* ignore */
  }
  return 'light'
}

function applyThemeToDocument(theme: AppTheme) {
  const root = document.documentElement
  root.classList.remove('app-theme-light', 'app-theme-dark', 'app-theme-eye')
  root.classList.add(APP_THEME_CLASSES[theme])
  root.setAttribute('data-theme', theme)
}

export function useAppTheme() {
  const [theme, setThemeState] = useState<AppTheme>(() => readStoredTheme())

  useEffect(() => {
    applyThemeToDocument(theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const setTheme = useCallback((next: AppTheme) => {
    setThemeState(next)
  }, [])

  const cycleTheme = useCallback(() => {
    setThemeState((prev) => {
      const idx = THEME_ORDER.indexOf(prev)
      return THEME_ORDER[(idx + 1) % THEME_ORDER.length]
    })
  }, [])

  return { theme, setTheme, cycleTheme }
}
