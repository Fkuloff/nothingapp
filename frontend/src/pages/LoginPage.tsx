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
    <div className="auth-layout">
      <div className="auth-panel">
        <div className="auth-panel__header">
          <div className="auth-logo">
            <span className="logo-dot" />
            <span className="logo-text">Pulse Messenger</span>
          </div>
          <p className="auth-subtitle">Войдите, чтобы продолжить общение и видеть все ваши диалоги.</p>
        </div>

        {error && (
          <div className="alert alert-danger auth-alert" role="alert">
            {error}
          </div>
        )}

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="auth-label" htmlFor="username">
            Имя пользователя
          </label>
          <input
            id="username"
            type="text"
            className="form-control auth-input"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            placeholder="Например, neo"
            required
            autoFocus
          />

          <label className="auth-label" htmlFor="password">
            Пароль
          </label>
          <input
            id="password"
            type="password"
            className="form-control auth-input"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="••••••••"
            required
          />

          <button type="submit" className="btn btn-primary w-100 auth-btn" disabled={submitting}>
            {submitting ? 'Вход...' : 'Войти'}
          </button>
        </form>

        <div className="auth-footer">
          <span>Нет аккаунта?</span>
          <Link to="/register" className="link-highlight">
            Зарегистрироваться
          </Link>
        </div>
      </div>

      <div className="auth-aside">
        <div className="auth-aside__card">
          <h3>Создано для диалогов</h3>
          <p>Легкий интерфейс, быстрые сообщения, отправка файлов и отзывчивая вёрстка.</p>
          <ul className="auth-highlights">
            <li>Живые обновления через WebSocket</li>
            <li>Темная тема, которая не режет глаз</li>
            <li>Заботливо оформленные состояния ошибок</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
