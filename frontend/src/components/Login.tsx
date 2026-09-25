import { useState, type FormEvent } from 'react'
import { IconBolt } from '../icons'

interface Props {
  onLogin: () => void
}

// Puerta de acceso simple para el flujo de demo de la sección 21 de la
// prueba ("Login → Dashboard → M-109 → ..."). No hay backend de
// autenticación en el alcance del MVP (ver la API mínima sugerida), así
// que esta pantalla valida solo que ambos campos estén diligenciados y
// deja pasar — el objetivo es que la app se sienta como un producto SaaS
// real y no una colección de pantallas sueltas, no implementar auth real.
export function Login({ onLogin }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!email.trim() || !password.trim()) {
      setError('Ingresa tu correo y contraseña.')
      return
    }
    setError(null)
    onLogin()
  }

  return (
    <div className="auth-shell">
      <form className="auth-card card" onSubmit={handleSubmit}>
        <div className="brand-mark" aria-hidden="true"><IconBolt /></div>
        <div>
          <h2>Bia Energy</h2>
          <p className="caption">Inicia sesión para gestionar tus medidores</p>
        </div>

        <div className="field">
          <label htmlFor="login-email">Correo</label>
          <input
            id="login-email"
            type="email"
            autoComplete="username"
            placeholder="tu@empresa.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>

        <div className="field">
          <label htmlFor="login-password">Contraseña</label>
          <input
            id="login-password"
            type="password"
            autoComplete="current-password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>

        {error && <p className="form-error">{error}</p>}

        <button type="submit" className="btn btn-primary btn-block">Iniciar sesión</button>
      </form>
    </div>
  )
}
