import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { useAuthContext } from '../features/auth/AuthContext'
import { endpoints } from '../shared/api/endpoints'
import { httpPost, setAuthToken } from '../shared/api/httpClient'
import type { AuthRegisterResponse } from '../shared/api/types'

export default function RegisterPage() {
  const [username, setUsername] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const navigate = useNavigate()
  const { refreshProfile } = useAuthContext()

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    const trimmedName = name.trim()
    if (trimmedName.length < 2) {
      setError('Имя должно содержать минимум 2 символа (без учёта пробелов)')
      setSubmitting(false)
      return
    }
    if (password.trim().length === 0) {
      setError('Пароль не может состоять только из пробелов')
      setSubmitting(false)
      return
    }

    try {
      const res = await httpPost<AuthRegisterResponse>(endpoints.auth.register, {
        username,
        name: trimmedName,
        password,
      })
      setAuthToken(res.token)
      await refreshProfile()
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка регистрации')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-centered">
      <div className="auth-card">
        <div className="auth-card__header">
          <span className="brand-text-lg">Nothing</span>
          <p className="auth-card__subtitle">Создайте аккаунт</p>
        </div>

        {error && (
          <div className="auth-error">
            {error}
          </div>
        )}

        <form className="auth-card__form" onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="username"
              required
              minLength={3}
              maxLength={20}
              autoFocus
              autoComplete="username"
            />
          </div>

          <div className="form-group">
            <label htmlFor="name">Имя</label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Как вас зовут"
              required
              minLength={2}
              maxLength={50}
              autoComplete="name"
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Минимум 6 символов"
              required
              minLength={6}
              autoComplete="new-password"
            />
          </div>

          <button type="submit" className="auth-card__submit" disabled={submitting}>
            {submitting ? 'Создаём...' : 'Создать аккаунт'}
          </button>
        </form>

        <div className="auth-card__footer">
          <span>Уже есть аккаунт?</span>
          <Link to="/login">Войти</Link>
        </div>
      </div>
    </div>
  )
}
