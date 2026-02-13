import { useState, useCallback } from 'react'
import { backupPrivateKey, restorePrivateKey } from '../../shared/crypto/keyExchange'
import { clearAllCryptoData } from '../../shared/crypto/keyStore'

type Props = {
  cryptoReady: boolean
  needsKeyRestore: boolean
  onKeysRestored: () => void
}

type KeyStatus = 'idle' | 'loading' | 'success' | 'error'

export function KeyManagement({ cryptoReady, needsKeyRestore, onKeysRestored }: Props) {
  const [backupPassword, setBackupPassword] = useState('')
  const [restorePassword, setRestorePassword] = useState('')
  const [backupStatus, setBackupStatus] = useState<KeyStatus>('idle')
  const [restoreStatus, setRestoreStatus] = useState<KeyStatus>('idle')
  const [statusMessage, setStatusMessage] = useState('')
  const [showResetConfirm, setShowResetConfirm] = useState(false)

  const handleBackup = useCallback(async () => {
    if (!backupPassword || backupPassword.length < 6) {
      setStatusMessage('Пароль должен быть не менее 6 символов')
      setBackupStatus('error')
      return
    }

    setBackupStatus('loading')
    setStatusMessage('')

    try {
      await backupPrivateKey(backupPassword)
      setBackupStatus('success')
      setStatusMessage('Ключ успешно сохранён на сервере')
      setBackupPassword('')
    } catch (err) {
      console.error('Backup failed:', err)
      setBackupStatus('error')
      setStatusMessage('Не удалось сохранить ключ')
    }
  }, [backupPassword])

  const handleRestore = useCallback(async () => {
    if (!restorePassword) {
      setStatusMessage('Введите пароль')
      setRestoreStatus('error')
      return
    }

    setRestoreStatus('loading')
    setStatusMessage('')

    try {
      const success = await restorePrivateKey(restorePassword)
      if (success) {
        setRestoreStatus('success')
        setStatusMessage('Ключи успешно восстановлены')
        setRestorePassword('')
        onKeysRestored()
      } else {
        setRestoreStatus('error')
        setStatusMessage('Неверный пароль или повреждённый бэкап')
      }
    } catch (err) {
      console.error('Restore failed:', err)
      setRestoreStatus('error')
      setStatusMessage('Ошибка восстановления ключей')
    }
  }, [restorePassword, onKeysRestored])

  const handleResetKeys = useCallback(async () => {
    await clearAllCryptoData()
    setShowResetConfirm(false)
    setStatusMessage('Ключи удалены. Перезагрузите страницу.')
    setBackupStatus('idle')
    setRestoreStatus('idle')
  }, [])

  const hasKeys = cryptoReady && !needsKeyRestore

  return (
    <div className="key-management">
      <div className="key-management__status">
        <span className="key-management__status-label">Ключи шифрования</span>
        <span className={`key-management__status-badge ${hasKeys ? 'active' : 'inactive'}`}>
          {hasKeys ? 'Активны' : needsKeyRestore ? 'Требуется восстановление' : 'Не настроены'}
        </span>
      </div>

      {needsKeyRestore && (
        <div className="key-management__section">
          <p className="key-management__hint">
            На сервере найден бэкап ключей. Введите пароль для восстановления.
          </p>
          <div className="key-management__input-group">
            <input
              type="password"
              className="form-control"
              placeholder="Пароль от бэкапа"
              value={restorePassword}
              onChange={(e) => setRestorePassword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleRestore()}
            />
            <button
              className="key-management__btn key-management__btn--primary"
              onClick={handleRestore}
              disabled={restoreStatus === 'loading'}
            >
              {restoreStatus === 'loading' ? 'Восстановление...' : 'Восстановить'}
            </button>
          </div>
        </div>
      )}

      {hasKeys && (
        <div className="key-management__section">
          <p className="key-management__hint">
            Создайте бэкап для доступа к сообщениям с других устройств.
          </p>
          <div className="key-management__input-group">
            <input
              type="password"
              className="form-control"
              placeholder="Придумайте пароль (мин. 6 символов)"
              value={backupPassword}
              onChange={(e) => setBackupPassword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleBackup()}
            />
            <button
              className="key-management__btn key-management__btn--primary"
              onClick={handleBackup}
              disabled={backupStatus === 'loading'}
            >
              {backupStatus === 'loading' ? 'Сохранение...' : 'Сохранить бэкап'}
            </button>
          </div>
        </div>
      )}

      {statusMessage && (
        <div
          className={`key-management__message ${
            backupStatus === 'error' || restoreStatus === 'error' ? 'error' : 'success'
          }`}
        >
          {statusMessage}
        </div>
      )}

      {hasKeys && (
        <div className="key-management__section">
          {!showResetConfirm ? (
            <button
              className="key-management__btn key-management__btn--danger-text"
              onClick={() => setShowResetConfirm(true)}
            >
              Сбросить ключи
            </button>
          ) : (
            <div className="key-management__confirm">
              <p className="key-management__warning">
                Все локальные ключи будут удалены. Без бэкапа вы потеряете доступ к зашифрованным сообщениям.
              </p>
              <div className="key-management__confirm-actions">
                <button
                  className="key-management__btn key-management__btn--danger"
                  onClick={handleResetKeys}
                >
                  Удалить ключи
                </button>
                <button
                  className="key-management__btn"
                  onClick={() => setShowResetConfirm(false)}
                >
                  Отмена
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
