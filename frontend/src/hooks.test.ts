import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MOBILE_BREAKPOINT, useMediaQuery, useTheme } from './hooks'

function makeMatchMedia(initialMatches: boolean) {
  let matches = initialMatches
  let listener: (() => void) | null = null
  const mql = {
    get matches() { return matches },
    media: MOBILE_BREAKPOINT,
    onchange: null,
    addEventListener: (_event: string, cb: () => void) => { listener = cb },
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }
  return {
    mql: mql as unknown as MediaQueryList,
    trigger(next: boolean) {
      matches = next
      listener?.()
    },
  }
}

describe('useMediaQuery', () => {
  afterEach(() => vi.restoreAllMocks())

  it('reflects the current match state on mount', () => {
    const { mql } = makeMatchMedia(true)
    vi.spyOn(window, 'matchMedia').mockReturnValue(mql)
    const { result } = renderHook(() => useMediaQuery(MOBILE_BREAKPOINT))
    expect(result.current).toBe(true)
  })

  it('updates when the underlying media query changes', () => {
    const { mql, trigger } = makeMatchMedia(false)
    vi.spyOn(window, 'matchMedia').mockReturnValue(mql)
    const { result } = renderHook(() => useMediaQuery(MOBILE_BREAKPOINT))
    expect(result.current).toBe(false)

    act(() => trigger(true))
    expect(result.current).toBe(true)
  })
})

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })
  afterEach(() => vi.restoreAllMocks())

  it('defaults to the stored theme when one is saved', () => {
    localStorage.setItem('bia-theme', 'dark')
    const { result } = renderHook(() => useTheme())
    expect(result.current[0]).toBe('dark')
  })

  it('falls back to a dark system preference when nothing is stored', () => {
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: true } as MediaQueryList)
    const { result } = renderHook(() => useTheme())
    expect(result.current[0]).toBe('dark')
  })

  it('falls back to light when the system has no dark preference and nothing is stored', () => {
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    const { result } = renderHook(() => useTheme())
    expect(result.current[0]).toBe('light')
  })

  it('falls back to system preference when reading localStorage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
    const { result } = renderHook(() => useTheme())
    expect(result.current[0]).toBe('light')
  })

  it('toggles the theme, persisting it and reflecting it on <html data-theme>', () => {
    localStorage.setItem('bia-theme', 'light')
    const { result } = renderHook(() => useTheme())
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')

    act(() => result.current[1]())

    expect(result.current[0]).toBe('dark')
    expect(localStorage.getItem('bia-theme')).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('does not throw when persisting the theme fails', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('blocked') })
    const { result } = renderHook(() => useTheme())
    expect(() => act(() => result.current[1]())).not.toThrow()
  })
})
