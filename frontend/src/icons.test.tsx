import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import * as Icons from './icons'

// Every icon is a trivial function component that returns a hand-drawn SVG
// with a default size. Rendering each one once, plus a custom size, is
// enough to exercise every line — there's no branching logic to speak of.
const ICON_NAMES = Object.keys(Icons) as (keyof typeof Icons)[]

describe('icon set', () => {
  it.each(ICON_NAMES)('%s renders an svg at its default size', (name) => {
    const Icon = Icons[name]
    const { container } = render(<Icon />)
    const svg = container.querySelector('svg')
    expect(svg).toBeTruthy()
  })

  it.each(ICON_NAMES)('%s honors a custom size prop', (name) => {
    const Icon = Icons[name]
    const { container } = render(<Icon size={40} />)
    const svg = container.querySelector('svg')
    expect(svg?.getAttribute('width')).toBe('40')
    expect(svg?.getAttribute('height')).toBe('40')
  })
})
