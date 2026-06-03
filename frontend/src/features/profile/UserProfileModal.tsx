import { useCallback,useEffect, useState } from 'react'

import { removeContact } from '../../shared/api/contactsApi'
import { endpoints } from '../../shared/api/endpoints'
import { httpGet, httpPost, resolveApiUrl } from '../../shared/api/httpClient'
import type { UserProfile } from '../../shared/api/types'
import { CloseIcon } from '../../shared/components/Icons'
import { useToast } from '../../shared/components/ToastContext'
import { useAndroidBack } from '../../shared/hooks/useAndroidBack'
import { useConfirmAction } from '../../shared/hooks/useConfirmAction'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  isOpen: boolean
  onClose: () => void
  userId: number
  username?: string
  avatarUrl?: string | null
  isOnline?: boolean
}

export function UserProfileModal({ isOpen, onClose, userId, username, avatarUrl, isOnline }: Props) {
  const { showToast } = useToast()
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })
  useAndroidBack(() => { onClose(); return true }, isOpen)
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [addingContact, setAddingContact] = useState(false)
  const { confirming: confirmingRemove, startConfirm, cancelConfirm } = useConfirmAction()

  useEffect(() => {
    if (!isOpen || !userId) return

    async function loadProfile() {
      setLoading(true)
      try {
        const data = await httpGet<UserProfile>(endpoints.profile(userId))
        setProfile(data)
      } catch (err) {
        console.error('Failed to load profile:', err)
        showToast('Не удалось загрузить профиль', 'error')
      } finally {
        setLoading(false)
      }
    }

    loadProfile()
  }, [isOpen, userId, showToast])

  // Reset confirmation state when modal closes
  useEffect(() => {
    if (!isOpen) cancelConfirm()
  }, [isOpen, cancelConfirm])

  const handleRemoveContact = useCallback(async () => {
    if (!profile?.id) return

    try {
      await removeContact(profile.id)
      setProfile((prev) => (prev ? { ...prev, is_contact: false } : prev))
      cancelConfirm()
      showToast('Удалено из контактов', 'success')
    } catch (err) {
      showToast('Ошибка: ' + (err instanceof Error ? err.message : 'Не удалось удалить контакт'), 'error')
    }
  }, [profile?.id, showToast, cancelConfirm])

  const handleAddContact = async () => {
    if (!profile?.id) return

    setAddingContact(true)
    try {
      await httpPost(endpoints.contacts.add(profile.id), {})
      setProfile((prev) => (prev ? { ...prev, is_contact: true } : prev))
      showToast('Добавлено в контакты', 'success')
    } catch {
      showToast('Не удалось добавить в контакты', 'error')
    } finally {
      setAddingContact(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="profile-modal-backdrop" onClick={handleBackdropClick}>
      <div className="profile-modal user-profile-modal" role="dialog" aria-modal="true">
        <div className="profile-modal__header-actions">
          <button className="profile-modal__action-btn" onClick={onClose} aria-label="Закрыть">
            <CloseIcon />
          </button>
        </div>

        {loading ? (
          <div className="profile-modal__loading">
            <span className="text-muted">Загрузка...</span>
          </div>
        ) : (
          <>
            <div className="profile-modal__avatar-wrapper user-profile-modal__avatar-wrapper">
              <img
                src={resolveApiUrl(profile?.avatar_url || avatarUrl) || '/img/default-avatar.svg'}
                alt="avatar"
                className="profile-modal__avatar"
              />
            </div>

            <div className="profile-modal__info">
              <h2 className="profile-modal__name">{profile?.name || username}</h2>
              <span className="profile-modal__username">@{profile?.username || username}</span>
              {isOnline !== undefined && (
                <div className="user-profile-modal__status">
                  <span className={`dot ${isOnline ? 'online' : 'offline'}`} />
                  <span>{isOnline ? 'В сети' : 'Не в сети'}</span>
                </div>
              )}
            </div>

            {profile && !profile.is_contact && (
              <div className="profile-modal__actions">
                <button
                  className="btn btn-primary"
                  onClick={handleAddContact}
                  disabled={addingContact}
                >
                  {addingContact ? 'Добавление...' : 'Добавить в контакты'}
                </button>
              </div>
            )}

            {profile?.is_contact && (
              confirmingRemove ? (
                <div className="user-profile-modal__confirm-remove">
                  <span className="user-profile-modal__confirm-text">Удалить из контактов?</span>
                  <div className="user-profile-modal__confirm-actions">
                    <button
                      onClick={handleRemoveContact}
                      className="contact-card__confirm-btn contact-card__confirm-btn--delete"
                    >
                      Удалить
                    </button>
                    <button
                      onClick={cancelConfirm}
                      className="contact-card__confirm-btn contact-card__confirm-btn--cancel"
                    >
                      Отмена
                    </button>
                  </div>
                </div>
              ) : (
                <div
                  className="user-profile-modal__contact-badge user-profile-modal__contact-badge--clickable"
                  onClick={startConfirm}
                  role="button"
                  tabIndex={0}
                  title="Нажмите, чтобы удалить из контактов"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                  <span>В контактах</span>
                </div>
              )
            )}
          </>
        )}
      </div>
    </div>
  )
}
