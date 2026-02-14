import { useEffect, useState, useRef } from 'react'
import { useAuthContext } from '../auth/AuthContext'
import { httpPost, httpPut } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'
import { useToast } from '../../shared/components/ToastContext'
import type { AvatarUploadResponse } from '../../shared/api/types'

type Props = {
  isOpen: boolean
  onClose: () => void
}

export function ProfileModal({ isOpen, onClose }: Props) {
  const { user, refreshProfile } = useAuthContext()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [name, setName] = useState('')
  const [saving, setSaving] = useState(false)
  const { showToast } = useToast()
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })

  useEffect(() => {
    if (user) {
      setName(user.name || '')
    }
  }, [user])

  useEffect(() => {
    if (!isOpen) {
      setIsEditing(false)
    }
  }, [isOpen])

  const handleAvatarClick = () => {
    fileInputRef.current?.click()
  }

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const formData = new FormData()
    formData.append('avatar', file)

    try {
      await httpPost<AvatarUploadResponse>(endpoints.avatar.upload, formData)
      await refreshProfile()
      showToast('Аватар успешно загружен', 'success')
    } catch (err) {
      console.error('Failed to upload avatar:', err)
      showToast('Не удалось загрузить аватар', 'error')
    } finally {
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleSave = async () => {
    if (!name.trim()) return
    setSaving(true)
    try {
      await httpPut(endpoints.profile(), { name: name.trim() })
      await refreshProfile()
      setIsEditing(false)
    } catch (err) {
      console.error('Failed to update profile:', err)
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    setName(user?.name || '')
    setIsEditing(false)
  }

  if (!isOpen) return null

  return (
    <div className="profile-modal-backdrop" onClick={handleBackdropClick}>
      <div className="profile-modal" role="dialog" aria-modal="true">
        <div className="profile-modal__header-actions">
          <button
            className="profile-modal__action-btn"
            onClick={() => setIsEditing(true)}
            aria-label="Редактировать"
            disabled={isEditing}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
            </svg>
          </button>
          <button className="profile-modal__action-btn" onClick={onClose} aria-label="Закрыть">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="profile-modal__avatar-wrapper" onClick={handleAvatarClick}>
          <img
            src={user?.avatar_url || '/img/default-avatar.svg'}
            alt="avatar"
            className="profile-modal__avatar"
          />
          <div className="profile-modal__avatar-overlay">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
              <circle cx="12" cy="13" r="4" />
            </svg>
          </div>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleAvatarChange}
          />
        </div>

        <div className="profile-modal__info">
          {isEditing ? (
            <input
              type="text"
              className="profile-modal__name-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Имя"
              autoFocus
            />
          ) : (
            <h2 className="profile-modal__name">{user?.name || user?.username}</h2>
          )}
          <span className="profile-modal__username">@{user?.username}</span>
        </div>

        {isEditing && (
          <div className="profile-modal__actions">
            <button
              className="btn btn-primary"
              onClick={handleSave}
              disabled={saving || !name.trim()}
            >
              {saving ? 'Сохранение...' : 'Сохранить'}
            </button>
            <button className="btn btn-secondary" onClick={handleCancel} disabled={saving}>
              Отмена
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
