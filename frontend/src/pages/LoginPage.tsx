import { useState, type FormEvent } from 'react'
import { useAuth } from '../auth'

export default function LoginPage() {
  const { login, register } = useAuth()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      if (mode === 'login') await login(email, password)
      else await register(email, displayName, password)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="login-card" onSubmit={submit}>
        <h1>
          Hef<span>ai</span>
        </h1>
        <p className="page-sub">Dit byggeprojekt, fra idé til færdig byggesag.</p>
        <div className="field">
          <label>E-mail</label>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        </div>
        {mode === 'register' && (
          <div className="field">
            <label>Navn</label>
            <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </div>
        )}
        <div className="field">
          <label>Adgangskode</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </div>
        {error && <p className="error-text">{error}</p>}
        <button className="btn" disabled={busy}>
          {mode === 'login' ? 'Log ind' : 'Opret konto'}
        </button>
        <p style={{ marginTop: 14, fontSize: 13 }}>
          {mode === 'login' ? (
            <>
              Ny her?{' '}
              <a href="#" onClick={(e) => (e.preventDefault(), setMode('register'))}>
                Opret konto
              </a>
            </>
          ) : (
            <>
              Har du en konto?{' '}
              <a href="#" onClick={(e) => (e.preventDefault(), setMode('login'))}>
                Log ind
              </a>
            </>
          )}
        </p>
      </form>
    </div>
  )
}
