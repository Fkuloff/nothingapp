import { useEffect, useState } from 'react'
import { useParams, Link, useOutletContext } from 'react-router-dom'
import { useAuthContext } from '../features/auth/AuthContext'
import { httpGet, httpPost } from '../shared/api/httpClient'
import type { UserProfile, AvatarUploadResponse } from '../shared/api/types'
import { endpoints } from '../shared/api/endpoints'
import { useToast } from '../shared/components/ToastContext'
import { addContact, removeContact } from '../shared/api/contactsApi'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import type { OutletContextType } from '../App'

export default function ProfilePage() {
  const { setMenuOpen } = useOutletContext<OutletContextType>()
  const { userId } = useParams<{ userId?: string }>()
  const { user: currentUser, refreshProfile } = useAuthContext()
  const { showToast } = useToast()
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  const isOwnProfile = !userId || userId === String(currentUser?.id)
  const displayUser = isOwnProfile ? currentUser : profile

  useEffect(() => {
    async function loadProfile() {
      if (isOwnProfile && currentUser) {
        setProfile(currentUser)
        setLoading(false)
        return
      }

      if (!userId) {
        setLoading(false)
        return
      }

      try {
        setLoading(true)
        const data = await httpGet<UserProfile>(endpoints.profile(userId))
        setProfile(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Не удалось загрузить профиль')
      } finally {
        setLoading(false)
      }
    }

    loadProfile()
  }, [userId, currentUser, isOwnProfile])

  const handleAvatarUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
    if (!allowedTypes.includes(file.type)) {
      showToast('Выберите изображение (JPEG, PNG, GIF или WebP)', 'warning')
      return
    }

    if (file.size > 10 * 1024 * 1024) {
      showToast('Файл слишком большой. Максимум 10MB', 'warning')
      return
    }

    const formData = new FormData()
    formData.append('avatar', file)

    try {
      setUploading(true)
      const response = await httpPost<AvatarUploadResponse>(endpoints.avatar.upload, formData)

      if (response.success && currentUser) {
        await refreshProfile()
        setProfile((prev) => prev ? { ...prev, avatar_url: response.avatar_url } : prev)
        showToast('Аватар обновлён', 'success')
      }
    } catch (err) {
      showToast('Ошибка загрузки аватара: ' + (err instanceof Error ? err.message : 'Неизвестная ошибка'), 'error')
    } finally {
      setUploading(false)
    }
  }

  const handleAddContact = async () => {
    if (!profile?.id) return

    try {
      await addContact(profile.id)
      showToast('Добавлено в контакты', 'success')

      // Update local state to reflect contact status
      setProfile((prev) => (prev ? { ...prev, is_contact: true } : prev))
    } catch (err) {
      showToast('Ошибка: ' + (err instanceof Error ? err.message : 'Не удалось добавить контакт'), 'error')
    }
  }

  const handleRemoveContact = async () => {
    if (!profile?.id) return

    try {
      await removeContact(profile.id)
      showToast('Удалено из контактов', 'success')
      setProfile((prev) => (prev ? { ...prev, is_contact: false } : prev))
    } catch (err) {
      showToast('Ошибка: ' + (err instanceof Error ? err.message : 'Не удалось удалить контакт'), 'error')
    }
  }

  if (loading) {
    return (
      <div className="page-container">
        <div className="page-header">
          {isOwnProfile ? (
            <HamburgerButton onClick={() => setMenuOpen(true)} />
          ) : (
            <Link to="/" className="back-link">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M19 12H5M12 19l-7-7 7-7" />
              </svg>
              Назад
            </Link>
          )}
          <h2>Профиль</h2>
        </div>
        <div className="page-content text-center text-muted">
          Загружаем профиль...
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="page-container">
        <div className="page-header">
          <Link to="/" className="back-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Назад
          </Link>
          <h2>Ошибка</h2>
        </div>
        <div className="page-content">
          <div className="alert alert-danger">{error}</div>
        </div>
      </div>
    )
  }

  if (!displayUser) {
    return (
      <div className="page-container">
        <div className="page-header">
          <Link to="/" className="back-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Назад
          </Link>
          <h2>Профиль</h2>
        </div>
        <div className="page-content">
          <div className="alert alert-warning">Профиль не найден</div>
        </div>
      </div>
    )
  }

  return (
    <div className="page-container">
      {/* Header */}
      <div className="page-header">
        {isOwnProfile ? (
          <HamburgerButton onClick={() => setMenuOpen(true)} />
        ) : (
          <Link to="/" className="back-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            Назад
          </Link>
        )}
        <h2>{isOwnProfile ? 'Мой профиль' : 'Профиль'}</h2>
      </div>

      {/* Content */}
      <div className="page-content">
        <div className="profile-hero glassy">
          <div className="profile-hero__left">
            <div className="profile-avatar">
              {isOwnProfile ? (
                <div className="avatar-upload-container">
                  <span className="avatar avatar-xl" id="profile-avatar">
                    <img src={displayUser.avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
                  </span>
                  <label htmlFor="avatar-input" className="avatar-upload-overlay" title="Загрузить аватар">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
                      <circle cx="12" cy="13" r="4" />
                    </svg>
                  </label>
                  <input
                    type="file"
                    id="avatar-input"
                    accept="image/jpeg,image/png,image/gif,image/webp"
                    className="hidden"
                    onChange={handleAvatarUpload}
                    disabled={uploading}
                  />
                </div>
              ) : (
                <span className="avatar avatar-xl">
                  <img src={displayUser.avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
                </span>
              )}
            </div>
            <div className="profile-meta">
              <h1 className="profile-name">{displayUser.name}</h1>
              <p className="profile-username">@{displayUser.username}</p>
            </div>
          </div>
          <div className="profile-hero__right">
            <div className="profile-actions">
              {!isOwnProfile && (
                profile?.is_contact ? (
                  <button
                    onClick={handleRemoveContact}
                    className="btn btn-outline-danger"
                  >
                    Удалить из контактов
                  </button>
                ) : (
                  <button
                    onClick={handleAddContact}
                    className="btn btn-outline-light"
                  >
                    Добавить в контакты
                  </button>
                )
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
