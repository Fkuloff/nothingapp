import { useState } from 'react'

import { TIMEOUT_OPTIONS } from '../../shared/appLock'
import { CheckIcon, ChevronDownIcon } from '../../shared/components/Icons'
import { useAppLock } from './AppLockContext'

/**
 * Settings → Конфиденциальность block (Android only — the caller gates on
 * platform): biometric login switch + custom autolock interval picker. The
 * native <select> renders as an OS dialog that ignores the app theme, so the
 * picker is a hand-rolled expandable list instead.
 */
export function AppLockSettings() {
  const { enabled, biometricsAvailable, timeoutMs, enable, disable, setTimeoutMs } = useAppLock()
  const [busy, setBusy] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)

  const handleToggle = async () => {
    if (busy) return
    if (enabled) {
      setPickerOpen(false)
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

  const currentLabel = TIMEOUT_OPTIONS.find((opt) => opt.value === timeoutMs)?.label ?? '—'

  return (
    <>
      <div className="settings-modal__option">
        <div>
          <div className="settings-modal__option-label">Вход по биометрии</div>
          <div className="settings-modal__option-hint">
            {biometricsAvailable || enabled
              ? 'Блокировать приложение в фоне'
              : 'Биометрия недоступна на этом устройстве'}
          </div>
        </div>
        <label className="toggle-switch">
          <input
            type="checkbox"
            checked={enabled}
            disabled={busy || (!biometricsAvailable && !enabled)}
            onChange={() => void handleToggle()}
          />
          <span className="toggle-slider" />
        </label>
      </div>

      {enabled && (
        <div className="applock-picker">
          <button
            type="button"
            className="applock-picker__trigger"
            aria-expanded={pickerOpen}
            onClick={() => setPickerOpen((open) => !open)}
          >
            <span className="settings-modal__option-label">Автоблокировка</span>
            <span className="applock-picker__value">
              {currentLabel}
              <ChevronDownIcon
                size={16}
                className={`applock-picker__chevron${pickerOpen ? ' applock-picker__chevron--open' : ''}`}
              />
            </span>
          </button>

          {pickerOpen && (
            <div className="applock-picker__options" role="radiogroup" aria-label="Автоблокировка">
              {TIMEOUT_OPTIONS.map((opt) => {
                const selected = opt.value === timeoutMs
                return (
                  <button
                    key={opt.value}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    className={`applock-picker__option${selected ? ' applock-picker__option--selected' : ''}`}
                    onClick={() => {
                      setTimeoutMs(opt.value)
                      setPickerOpen(false)
                    }}
                  >
                    <span>{opt.label}</span>
                    {selected && <CheckIcon size={16} />}
                  </button>
                )
              })}
            </div>
          )}
        </div>
      )}
    </>
  )
}
