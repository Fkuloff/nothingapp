import { useEffect, useState } from 'react'
import { httpGet, httpPost } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'
import { useToast } from '../../shared/components/ToastContext'
import type { UserProfile } from '../../shared/api/types'

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
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [addingContact, setAddingContact] = useState(false)

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
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
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
                src={profile?.avatar_url || avatarUrl || '/img/default-avatar.svg'}
                alt="avatar"
                className="profile-modal__avatar"
              />
              {isOnline !== undefined && (
                <span className={`user-profile-modal__status-dot ${isOnline ? 'online' : 'offline'}`} />
              )}
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
              <div className="user-profile-modal__contact-badge">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
                <span>В контактах</span>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
