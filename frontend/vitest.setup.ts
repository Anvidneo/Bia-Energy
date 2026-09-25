// Runs once per test file. React Testing Library doesn't auto-unmount
// between tests unless something wires this up — without it, renders from
// earlier tests in the same file pile up in document.body and later
// queries like getByText start matching more than one element.
import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

afterEach(() => {
  cleanup()
})

// jsdom implements neither API. Both are used by real code (useMediaQuery/
// useTheme call matchMedia; recharts' ResponsiveContainer uses
// ResizeObserver to size the chart) — stub them so components that touch
// them can mount in tests instead of throwing "not implemented".
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(), // deprecated, some libs still call it
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

if (typeof window !== 'undefined' && !window.ResizeObserver) {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  window.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver
}
