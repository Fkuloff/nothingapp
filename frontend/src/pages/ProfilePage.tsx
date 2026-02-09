import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useAuthContext } from '../features/auth/AuthContext'
import { httpGet, httpPost } from '../shared/api/httpClient'
import type { UserProfile, AvatarUploadResponse } from '../shared/api/types'
import { endpoints } from '../shared/api/endpoints'
import { useToast } from '../shared/components/ToastContext'

export default function ProfilePage() {
  const { userId } = useParams<{ userId?: string }>()
  const { user: currentUser } = useAuthContext()
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
        setProfile({ ...currentUser, avatar_url: response.avatar_url })
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
      await httpPost(endpoints.contacts.add(profile.id), {})
      showToast('Добавлено в контакты', 'success')
    } catch (err) {
      showToast('Ошибка: ' + (err instanceof Error ? err.message : 'Не удалось добавить контакт'), 'error')
    }
  }

  if (loading) {
    return (
      <div className="container mt-4 text-center text-muted">
        Загружаем профиль...
      </div>
    )
  }

  if (error) {
    return (
      <div className="container mt-4">
        <div className="alert alert-danger">{error}</div>
        <Link to="/" className="btn btn-secondary">Вернуться к чатам</Link>
      </div>
    )
  }

  if (!displayUser) {
    return (
      <div className="container mt-4">
        <div className="alert alert-warning">Профиль не найден</div>
        <Link to="/" className="btn btn-secondary">Вернуться к чатам</Link>
      </div>
    )
  }

  return (
    <div className="container mt-4 profile-layout">
      <div className="profile-hero glassy">
        <div className="profile-hero__left">
          <div className="profile-avatar">
            {isOwnProfile ? (
              <div className="avatar-upload-container">
                <span className="avatar avatar-xl" id="profile-avatar">
                  <img src={displayUser.avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
                </span>
                <label htmlFor="avatar-input" className="avatar-upload-overlay" title="Загрузить аватар">
                  <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12" />
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
            <p className="eyebrow">Профиль</p>
            <h1 className="profile-name">{displayUser.name}</h1>
            <p className="profile-username">@{displayUser.username}</p>
            <p className="profile-phone">{displayUser.phone}</p>
          </div>
        </div>
        <div className="profile-hero__right">
          <div className="chip">{isOwnProfile ? 'Это ваш аккаунт' : 'Пользователь'}</div>
          <div className="profile-actions">
            {!isOwnProfile && (
              <button onClick={handleAddContact} className="btn btn-outline-light">
                Добавить в контакты
              </button>
            )}
            <Link to="/" className="btn btn-primary">
              Назад к чатам
            </Link>
          </div>
        </div>
      </div>

      <div className="profile-grid">
        <div className="profile-tile glassy">
          <p className="eyebrow">Основное</p>
          <div className="meta-row">
            <span className="meta-label">Имя пользователя</span>
            <span className="meta-value">@{displayUser.username}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Телефон</span>
            <span className="meta-value">{displayUser.phone}</span>
          </div>
        </div>

        <div className="profile-tile glassy">
          <p className="eyebrow">Состояние</p>
          <p className="text-muted mb-0">Все настройки аккаунта пока в разработке.</p>
        </div>
      </div>
    </div>
  )
}
