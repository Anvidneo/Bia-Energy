import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Login } from './Login'

describe('Login', () => {
  it('renders the branded form', () => {
    render(<Login onLogin={() => {}} />)
    expect(screen.getByText('Bia Energy')).toBeTruthy()
    expect(screen.getByLabelText('Correo')).toBeTruthy()
    expect(screen.getByLabelText('Contraseña')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Iniciar sesión' })).toBeTruthy()
  })

  it('shows a validation error and does not log in when fields are empty', () => {
    const onLogin = vi.fn()
    render(<Login onLogin={onLogin} />)
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    expect(screen.getByText('Ingresa tu correo y contraseña.')).toBeTruthy()
    expect(onLogin).not.toHaveBeenCalled()
  })

  it('logs in once both fields are filled in', () => {
    const onLogin = vi.fn()
    render(<Login onLogin={onLogin} />)
    fireEvent.change(screen.getByLabelText('Correo'), { target: { value: 'anvid@bia.energy' } })
    fireEvent.change(screen.getByLabelText('Contraseña'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    expect(onLogin).toHaveBeenCalledTimes(1)
  })

  it('clears a prior error once the form is resubmitted with valid input', () => {
    const onLogin = vi.fn()
    render(<Login onLogin={onLogin} />)
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    expect(screen.getByText('Ingresa tu correo y contraseña.')).toBeTruthy()

    fireEvent.change(screen.getByLabelText('Correo'), { target: { value: 'anvid@bia.energy' } })
    fireEvent.change(screen.getByLabelText('Contraseña'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    expect(screen.queryByText('Ingresa tu correo y contraseña.')).toBeNull()
    expect(onLogin).toHaveBeenCalledTimes(1)
  })

  it('rejects whitespace-only input as empty', () => {
    const onLogin = vi.fn()
    render(<Login onLogin={onLogin} />)
    fireEvent.change(screen.getByLabelText('Correo'), { target: { value: '   ' } })
    fireEvent.change(screen.getByLabelText('Contraseña'), { target: { value: '   ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    expect(screen.getByText('Ingresa tu correo y contraseña.')).toBeTruthy()
    expect(onLogin).not.toHaveBeenCalled()
  })
})
