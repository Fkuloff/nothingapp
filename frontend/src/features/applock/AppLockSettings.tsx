import { useState } from 'react'

import { TIMEOUT_OPTIONS } from '../../shared/appLock'
import { useAppLock } from './AppLockContext'

/**
 * Settings → Конфиденциальность block (Android only — the caller gates on
 * platform): biometric login toggle + autolock interval selector.
 */
export function AppLockSettings() {
  const { enabled, biometricsAvailable, timeoutMs, enable, disable, setTimeoutMs } = useAppLock()
  const [busy, setBusy] = useState(false)

  const handleToggle = async () => {
    if (busy) return
    if (enabled) {
      disable()
      return
    }
    setBusy(true)
    try {
      await enable() // shows the system prompt; cancel keeps it off
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className="settings-modal__option">
        <span className="settings-modal__option-label">Вход по биометрии</span>
        {biometricsAvailable || enabled ? (
          <button
            className={`btn btn-sm ${enabled ? 'btn-success' : 'btn-outline-secondary'}`}
            onClick={() => void handleToggle()}
            disabled={busy}
          >
            {busy ? '...' : enabled ? 'Вкл' : 'Выкл'}
          </button>
        ) : (
          <span className="settings-modal__placeholder">Биометрия недоступна</span>
        )}
      </div>

      {enabled && (
        <div className="settings-modal__option">
          <span className="settings-modal__option-label">Автоблокировка</span>
          <select
            className="form-select form-select-sm"
            style={{ width: 'auto' }}
            value={timeoutMs}
            onChange={(e) => setTimeoutMs(Number(e.target.value))}
          >
            {TIMEOUT_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      )}
    </>
  )
}
