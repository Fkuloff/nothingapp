import { CloseIcon } from '../../shared/components/Icons'
import { PushToggle } from '../../shared/components/PushToggle'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: Props) {
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })

  if (!isOpen) return null

  return (
    <div className="settings-modal-backdrop" onClick={handleBackdropClick}>
      <div className="settings-modal" role="dialog" aria-modal="true">
        <div className="settings-modal__header">
          <h2 className="settings-modal__title">Настройки</h2>
          <button className="settings-modal__close" onClick={onClose} aria-label="Закрыть">
            <CloseIcon />
          </button>
        </div>

        <div className="settings-modal__content">
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
