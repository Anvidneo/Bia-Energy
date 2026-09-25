import { beforeEach, describe, expect, it, vi } from 'vitest'

const render = vi.fn()
const createRoot = vi.fn(() => ({ render }))

vi.mock('react-dom/client', () => ({ createRoot }))
vi.mock('./App.tsx', () => ({ default: () => null }))

describe('main', () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="root"></div>'
    vi.resetModules()
    createRoot.mockClear()
    render.mockClear()
  })

  it('mounts App into the #root element', async () => {
    await import('./main.tsx')
    const rootEl = document.getElementById('root')
    expect(createRoot).toHaveBeenCalledWith(rootEl)
    expect(render).toHaveBeenCalledTimes(1)
  })
})
