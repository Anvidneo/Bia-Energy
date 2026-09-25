import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Layout } from './Layout'

function setup(props: Partial<Parameters<typeof Layout>[0]> = {}) {
  const onNavigate = vi.fn()
  const onToggleTheme = vi.fn()
  const onSearch = vi.fn()
  const utils = render(
    <Layout
      screen="dashboard"
      onNavigate={onNavigate}
      theme="light"
      onToggleTheme={onToggleTheme}
      onSearch={onSearch}
      {...props}
    >
      <p>contenido</p>
    </Layout>,
  )
  return { ...utils, onNavigate, onToggleTheme, onSearch }
}

describe('Layout', () => {
  it('renders the title for the active screen and its children', () => {
    setup({ screen: 'anomalies' })
    expect(screen.getByRole('heading', { name: 'Anomalías' })).toBeTruthy()
    expect(screen.getByText('contenido')).toBeTruthy()
  })

  it('marks the active screen\'s nav button as current', () => {
    setup({ screen: 'meters' })
    const metersButtons = screen.getAllByLabelText('Medidores')
    expect(metersButtons.some((b) => b.getAttribute('aria-current') === 'page')).toBe(true)
  })

  it('calls onNavigate when a sidebar nav button is clicked', () => {
    const { onNavigate } = setup()
    fireEvent.click(screen.getAllByLabelText('Análisis')[0])
    expect(onNavigate).toHaveBeenCalledWith('analysis')
  })

  it('opens and closes the mobile nav panel', () => {
    setup()
    expect(screen.queryByLabelText('Cerrar menú')).toBeNull()

    fireEvent.click(screen.getByLabelText('Abrir menú'))
    expect(screen.getAllByLabelText('Cerrar menú').length).toBeGreaterThan(0)

    fireEvent.click(screen.getAllByLabelText('Cerrar menú')[0])
    expect(screen.queryByLabelText('Cerrar menú')).toBeNull()
  })

  it('closes the mobile nav panel after picking a nav item from it', () => {
    setup()
    fireEvent.click(screen.getByLabelText('Abrir menú'))
    expect(screen.getAllByLabelText('Cerrar menú').length).toBeGreaterThan(0)

    const dashboardButtons = screen.getAllByLabelText('Dashboard')
    fireEvent.click(dashboardButtons[dashboardButtons.length - 1])
    expect(screen.queryByLabelText('Cerrar menú')).toBeNull()
  })

  it('closes the mobile nav panel via its own close (X) button', () => {
    setup()
    fireEvent.click(screen.getByLabelText('Abrir menú'))
    const closeButtons = screen.getAllByLabelText('Cerrar menú')
    expect(closeButtons.length).toBeGreaterThan(1)

    // closeButtons[0] is the backdrop overlay; the panel's own X button is
    // the other one — click it specifically to cover that handler too.
    fireEvent.click(closeButtons[1])
    expect(screen.queryByLabelText('Cerrar menú')).toBeNull()
  })

  it('calls onToggleTheme from the desktop theme switch', () => {
    const { onToggleTheme } = setup({ theme: 'light' })
    fireEvent.click(screen.getByLabelText('Tema oscuro'))
    expect(onToggleTheme).toHaveBeenCalled()
  })

  it('does not call onToggleTheme when clicking the already-active theme button', () => {
    const { onToggleTheme } = setup({ theme: 'light' })
    fireEvent.click(screen.getByLabelText('Tema claro'))
    expect(onToggleTheme).not.toHaveBeenCalled()
  })

  it('calls onToggleTheme from the mobile header icon', () => {
    const { onToggleTheme } = setup({ theme: 'dark' })
    fireEvent.click(screen.getByLabelText('Cambiar a tema claro'))
    expect(onToggleTheme).toHaveBeenCalled()
  })

  it('submits the search box value via onSearch', () => {
    const { onSearch } = setup()
    const input = screen.getByPlaceholderText('Buscar medidor, anomalía...')
    fireEvent.change(input, { target: { value: 'M-110' } })
    fireEvent.submit(input.closest('form') as HTMLFormElement)
    expect(onSearch).toHaveBeenCalledWith('M-110')
  })

  it('does not throw when no onSearch handler is provided', () => {
    const onNavigate = vi.fn()
    const onToggleTheme = vi.fn()
    render(
      <Layout screen="dashboard" onNavigate={onNavigate} theme="light" onToggleTheme={onToggleTheme}>
        <p>contenido</p>
      </Layout>,
    )
    const input = screen.getByPlaceholderText('Buscar medidor, anomalía...')
    expect(() => fireEvent.submit(input.closest('form') as HTMLFormElement)).not.toThrow()
  })

  it('renders a plain, non-interactive account avatar when no onLogout is provided', () => {
    setup()
    const avatar = screen.getByLabelText('Cuenta')
    expect(avatar.tagName).toBe('DIV')
  })

  it('renders a clickable logout avatar and calls onLogout when provided', () => {
    const onLogout = vi.fn()
    setup({ onLogout })
    const avatar = screen.getByLabelText('Cerrar sesión')
    expect(avatar.tagName).toBe('BUTTON')
    fireEvent.click(avatar)
    expect(onLogout).toHaveBeenCalledTimes(1)
  })

  it('navigates to the dashboard when the sidebar logo is clicked', () => {
    const { onNavigate } = setup({ screen: 'anomalies' })
    const logos = screen.getAllByLabelText('Ir al dashboard')
    fireEvent.click(logos[0])
    expect(onNavigate).toHaveBeenCalledWith('dashboard')
  })

  it('navigates to the dashboard when the mobile header logo is clicked', () => {
    const { onNavigate } = setup({ screen: 'anomalies' })
    const logos = screen.getAllByLabelText('Ir al dashboard')
    fireEvent.click(logos[logos.length - 1])
    expect(onNavigate).toHaveBeenCalledWith('dashboard')
  })

  it('navigates to the dashboard and closes the mobile nav panel when its logo is clicked', () => {
    const { onNavigate } = setup({ screen: 'anomalies' })
    fireEvent.click(screen.getByLabelText('Abrir menú'))

    const logos = screen.getAllByLabelText('Ir al dashboard')
    expect(logos).toHaveLength(3)
    fireEvent.click(logos[1])

    expect(onNavigate).toHaveBeenCalledWith('dashboard')
    expect(screen.queryByLabelText('Cerrar menú')).toBeNull()
  })
})
