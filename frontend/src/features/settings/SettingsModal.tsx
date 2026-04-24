import { useCallback, useState } from 'react'

import { ArrowLeftIcon, BellIcon, CloseIcon, LockIcon } from '../../shared/components/Icons'
import { PushToggle } from '../../shared/components/PushToggle'
import { useAndroidBack } from '../../shared/hooks/useAndroidBack'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'
import { ChangePasswordForm } from './ChangePasswordForm'

type SettingsView = 'main' | 'notifications' | 'security'

const viewTitles: Record<SettingsView, string> = {
  main: 'Настройки',
  notifications: 'Уведомления',
  security: 'Конфиденциальность',
}

type Props = {
  isOpen: boolean
  onClose: () => void
}

export function SettingsModal({ isOpen, onClose }: Props) {
  const [activeView, setActiveView] = useState<SettingsView>('main')

  const closeModal = useCallback(() => {
    setActiveView('main')
    onClose()
  }, [onClose])

  const handleEscape = useCallback(() => {
    if (activeView !== 'main') {
      setActiveView('main')
    } else {
      closeModal()
    }
  }, [activeView, closeModal])

  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose: handleEscape })
  useAndroidBack(() => { handleEscape(); return true }, isOpen)

  if (!isOpen) return null

  return (
    <div className="settings-modal-backdrop" onClick={handleBackdropClick}>
      <div className="settings-modal" role="dialog" aria-modal="true">
        <div className="settings-modal__header">
          {activeView !== 'main' ? (
            <button
              className="settings-modal__close"
              onClick={() => setActiveView('main')}
              aria-label="Назад"
            >
              <ArrowLeftIcon size={20} />
            </button>
          ) : (
            <div />
          )}
          <h2 className="settings-modal__title">{viewTitles[activeView]}</h2>
          <button className="settings-modal__close" onClick={closeModal} aria-label="Закрыть">
            <CloseIcon />
          </button>
        </div>

        <div className="settings-modal__content">
          {activeView === 'main' && (
            <div className="settings-modal__list">
              <button
                className="settings-modal__list-item"
                onClick={() => setActiveView('notifications')}
              >
                <span className="settings-modal__list-icon">
                  <BellIcon size={20} />
                </span>
                <span>Уведомления</span>
              </button>

              <button
                className="settings-modal__list-item"
                onClick={() => setActiveView('security')}
              >
                <span className="settings-modal__list-icon">
                  <LockIcon size={20} />
                </span>
                <span>Конфиденциальность</span>
              </button>
            </div>
          )}

          {activeView === 'notifications' && (
            <div className="settings-modal__section">
              <div className="settings-modal__option">
                <span className="settings-modal__option-label">Push-уведомления</span>
                <PushToggle placeholderClass="settings-modal__placeholder" />
              </div>
            </div>
          )}

          {activeView === 'security' && (
            <div className="settings-modal__section">
              <ChangePasswordForm />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
