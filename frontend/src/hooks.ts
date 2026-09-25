import { useEffect, useState } from 'react'

// Matches the design's desktop (icon-rail + table) vs mobile
// (hamburger + stacked cards) breakpoint.
export const MOBILE_BREAKPOINT = '(max-width: 860px)'

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(query).matches : false,
  )

  useEffect(() => {
    const mql = window.matchMedia(query)
    const onChange = () => setMatches(mql.matches)
    onChange()
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [query])

  return matches
}

export type Theme = 'light' | 'dark'

export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(() => {
    try {
      const stored = localStorage.getItem('bia-theme')
      if (stored === 'light' || stored === 'dark') return stored
    } catch {
      // localStorage unavailable — fall back to system preference
    }
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    try {
      localStorage.setItem('bia-theme', theme)
    } catch {
      // best-effort persistence only
    }
  }, [theme])

  const toggle = () => setTheme((t) => (t === 'light' ? 'dark' : 'light'))
  return [theme, toggle]
}
