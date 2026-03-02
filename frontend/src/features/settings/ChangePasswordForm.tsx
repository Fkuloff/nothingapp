import { type FormEvent, useState } from 'react'

import { endpoints } from '../../shared/api/endpoints'
import { httpPut } from '../../shared/api/httpClient'

type Status = 'idle' | 'loading' | 'success' | 'error'

export function ChangePasswordForm() {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [status, setStatus] = useState<Status>('idle')
  const [errorMessage, setErrorMessage] = useState('')

  const canSubmit =
    status !== 'loading' &&
    oldPassword.length > 0 &&
    newPassword.length >= 6 &&
    newPassword === confirmPassword

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!canSubmit) return

    setStatus('loading')
    setErrorMessage('')

    try {
      await httpPut(endpoints.auth.changePassword, {
        old_password: oldPassword,
        new_password: newPassword,
      })
      setStatus('success')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setTimeout(() => setStatus('idle'), 3000)
    } catch (err) {
      setStatus('error')
      const msg = err instanceof Error ? err.message : ''
      if (msg.includes('Неверный текущий пароль')) {
        setErrorMessage('Неверный текущий пароль')
      } else {
        setErrorMessage('Не удалось сменить пароль')
      }
    }
  }

  return (
    <form className="change-password-form" onSubmit={handleSubmit}>
      <div className="change-password-form__field">
        <label htmlFor="old-password">Текущий пароль</label>
        <input
          id="old-password"
          type="password"
          className="form-control"
          value={oldPassword}
          onChange={(e) => setOldPassword(e.target.value)}
          autoComplete="current-password"
          disabled={status === 'loading'}
        />
      </div>

      <div className="change-password-form__field">
        <label htmlFor="new-password">Новый пароль</label>
        <input
          id="new-password"
          type="password"
          className="form-control"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          minLength={6}
          autoComplete="new-password"
          disabled={status === 'loading'}
        />
      </div>

      <div className="change-password-form__field">
        <label htmlFor="confirm-password">Подтвердите пароль</label>
        <input
          id="confirm-password"
          type="password"
          className="form-control"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          minLength={6}
          autoComplete="new-password"
          disabled={status === 'loading'}
        />
        {confirmPassword.length > 0 && newPassword !== confirmPassword && (
          <span className="change-password-form__hint">Пароли не совпадают</span>
        )}
      </div>

      {status === 'error' && (
        <div className="change-password-form__error">{errorMessage}</div>
      )}
      {status === 'success' && (
        <div className="change-password-form__success">Пароль успешно изменён</div>
      )}

      <button
        type="submit"
        className="btn btn-sm btn-primary change-password-form__submit"
        disabled={!canSubmit}
      >
        {status === 'loading' ? 'Сохранение...' : 'Сменить пароль'}
      </button>
    </form>
  )
}
