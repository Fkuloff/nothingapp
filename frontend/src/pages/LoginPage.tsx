import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { httpPost, setAuthToken } from '../shared/api/httpClient'
import { endpoints } from '../shared/api/endpoints'
import { useAuthContext } from '../features/auth/AuthContext'
import type { AuthLoginResponse } from '../shared/api/types'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const navigate = useNavigate()
  const { refreshProfile } = useAuthContext()

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const res = await httpPost<AuthLoginResponse>(endpoints.auth.login, { username, password })
      setAuthToken(res.token)
      await refreshProfile()
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка входа')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-centered">
      <div className="auth-card">
        <div className="auth-card__header">
          <span className="brand-text-lg">nothing</span>
          <p className="auth-card__subtitle">Войдите в аккаунт</p>
        </div>

        {error && (
          <div className="auth-error">
            {error}
          </div>
        )}

        <form className="auth-card__form" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Имя пользователя</label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="username"
              required
              autoFocus
              autoComplete="username"
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="••••••••"
              required
              autoComplete="current-password"
            />
          </div>

          <button type="submit" className="auth-card__submit" disabled={submitting}>
            {submitting ? 'Вход...' : 'Войти'}
          </button>
        </form>

        <div className="auth-card__footer">
          <span>Нет аккаунта?</span>
          <Link to="/register">Создать</Link>
        </div>
      </div>
    </div>
  )
}
