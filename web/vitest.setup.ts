import '@testing-library/jest-dom/vitest'

// ── Mock window.matchMedia (required by Ant Design) ──────────────────
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }),
})

// ── Mock window.getComputedStyle ─────────────────────────────────────
const originalGetComputedStyle = window.getComputedStyle
Object.defineProperty(window, 'getComputedStyle', {
  writable: true,
  value: (elt: Element) => {
    const styles = originalGetComputedStyle.call(window, elt)
    return {
      ...styles,
      getPropertyValue: (prop: string) => styles.getPropertyValue(prop) || '',
    }
  },
})

// ── Mock localStorage ────────────────────────────────────────────────
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = String(value)
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    },
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
  writable: true,
})

// ── Mock location for router tests ───────────────────────────────────
let mockLocation = new URL('http://localhost/')
Object.defineProperty(window, 'location', {
  get() {
    return mockLocation
  },
  set(newLocation: URL) {
    mockLocation = newLocation
  },
  configurable: true,
})
