import { useTheme } from '../../shared/hooks/useTheme'
import { PushToggle } from '../../shared/components/PushToggle'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: Props) {
  const { theme, setTheme } = useTheme()
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })

  if (!isOpen) return null

  return (
    <div className="settings-modal-backdrop" onClick={handleBackdropClick}>
      <div className="settings-modal" role="dialog" aria-modal="true">
        <div className="settings-modal__header">
          <h2 className="settings-modal__title">Настройки</h2>
          <button className="settings-modal__close" onClick={onClose} aria-label="Закрыть">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="settings-modal__content">
          <div className="settings-modal__section">
            <h3 className="settings-modal__section-title">Внешний вид</h3>
            <div className="settings-modal__option">
              <span className="settings-modal__option-label">Тема</span>
              <div className="settings-modal__theme-buttons">
                <button
                  className={`settings-modal__theme-btn ${theme === 'light' ? 'active' : ''}`}
                  onClick={() => setTheme('light')}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <circle cx="12" cy="12" r="5" />
                    <line x1="12" y1="1" x2="12" y2="3" />
                    <line x1="12" y1="21" x2="12" y2="23" />
                    <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                    <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                    <line x1="1" y1="12" x2="3" y2="12" />
                    <line x1="21" y1="12" x2="23" y2="12" />
                    <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                    <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
                  </svg>
                  Светлая
                </button>
                <button
                  className={`settings-modal__theme-btn ${theme === 'dark' ? 'active' : ''}`}
                  onClick={() => setTheme('dark')}
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
                  </svg>
                  Тёмная
                </button>
              </div>
            </div>
          </div>

          <div className="settings-modal__section">
            <h3 className="settings-modal__section-title">Уведомления</h3>
            <div className="settings-modal__option">
              <span className="settings-modal__option-label">Push-уведомления</span>
              <PushToggle placeholderClass="settings-modal__placeholder" />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
