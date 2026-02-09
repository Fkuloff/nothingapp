import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { httpPost, setAuthToken } from '../shared/api/httpClient'
import { endpoints } from '../shared/api/endpoints'
import { useAuthContext } from '../features/auth/AuthContext'
import type { AuthRegisterResponse } from '../shared/api/types'

export default function RegisterPage() {
  const [username, setUsername] = useState('')
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
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
      const res = await httpPost<AuthRegisterResponse>(endpoints.auth.register, {
        username,
        name,
        phone,
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
    <div className="auth-layout">
      <div className="auth-panel">
        <div className="auth-panel__header">
          <div className="auth-logo">
            <span className="logo-dot" />
            <span className="logo-text">Pulse Messenger</span>
          </div>
          <p className="auth-subtitle">Создайте аккаунт, чтобы начать новые диалоги и добавлять контакты.</p>
        </div>

        {error && (
          <div className="alert alert-danger auth-alert" role="alert">
            {error}
          </div>
        )}

        <form className="auth-form" onSubmit={handleSubmit}>
          <div className="auth-grid">
            <div>
              <label className="auth-label" htmlFor="username">
                Имя пользователя
              </label>
              <input
                id="username"
                type="text"
                className="form-control auth-input"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="@username"
                required
                minLength={3}
                maxLength={20}
                autoFocus
              />
            </div>
            <div>
              <label className="auth-label" htmlFor="name">
                Полное имя
              </label>
              <input
                id="name"
                type="text"
                className="form-control auth-input"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Как вас зовут"
                required
                minLength={2}
                maxLength={50}
              />
            </div>
          </div>

          <label className="auth-label" htmlFor="phone">
            Телефон
          </label>
          <input
            id="phone"
            type="tel"
            className="form-control auth-input"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            placeholder="+79991234567"
            required
          />
          <small className="text-muted d-block mb-3">Международный формат, например +79991234567</small>

          <label className="auth-label" htmlFor="password">
            Пароль
          </label>
          <input
            id="password"
            type="password"
            className="form-control auth-input"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="Минимум 6 символов"
            required
            minLength={6}
          />

          <button type="submit" className="btn btn-primary w-100 auth-btn" disabled={submitting}>
            {submitting ? 'Создаём...' : 'Зарегистрироваться'}
          </button>
        </form>

        <div className="auth-footer">
          <span>Уже есть аккаунт?</span>
          <Link to="/login" className="link-highlight">
            Войти
          </Link>
        </div>
      </div>

      <div className="auth-aside">
        <div className="auth-aside__card">
          <h3>Что внутри</h3>
          <p>Поддержка ответов, редактирования, удаления сообщений и загрузки вложений.</p>
          <ul className="auth-highlights">
            <li>Список диалогов с бейджами непрочитанного</li>
            <li>Профиль и загрузка аватара</li>
            <li>Продуманные состояния пустых экранов</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
