import { useState } from 'react'
import { useLocation,useNavigate } from 'react-router-dom'

import { createChat } from '../../shared/api/chatsApi'
import { removeContact } from '../../shared/api/contactsApi'
import type { UserListItem } from '../../shared/api/types'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'
import { useTheme } from '../../shared/hooks/useTheme'
import { useAuthContext } from '../auth/AuthContext'
import { ContactsModal } from '../contacts/ContactsModal'
import { ProfileModal } from '../profile/ProfileModal'
import { SettingsModal } from '../settings/SettingsModal'

type Props = {
  isOpen: boolean
  onClose: () => void
}

export function SlideMenu({ isOpen, onClose }: Props) {
  const { user, logout } = useAuthContext()
  const { theme, toggleTheme } = useTheme()
  const navigate = useNavigate()
  const location = useLocation()
  const [profileModalOpen, setProfileModalOpen] = useState(false)
  const [contactsModalOpen, setContactsModalOpen] = useState(false)
  const [settingsModalOpen, setSettingsModalOpen] = useState(false)
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })

  const handleLogout = async () => {
    await logout()
    navigate('/login')
    onClose()
  }

  const handleNavigate = (path: string) => {
    if (location.pathname !== path) {
      navigate(path)
    }
    onClose()
  }

  const isActive = (path: string) => location.pathname === path

  const handleOpenProfile = () => {
    setProfileModalOpen(true)
    onClose()
  }

  const handleOpenContacts = () => {
    setContactsModalOpen(true)
    onClose()
  }

  const handleSelectContact = async (contact: UserListItem) => {
    try {
      const chat = await createChat(contact.id)
      navigate(`/?chat=${chat.id}`)
    } catch (err) {
      console.error('Failed to create chat:', err)
    }
  }

  const handleRemoveContact = async (userId: number) => {
    await removeContact(userId)
  }

  const handleOpenSettings = () => {
    setSettingsModalOpen(true)
    onClose()
  }

  return (
    <>
      {/* Profile Modal */}
      <ProfileModal isOpen={profileModalOpen} onClose={() => setProfileModalOpen(false)} />

      {/* Contacts Modal */}
      <ContactsModal
        isOpen={contactsModalOpen}
        onClose={() => setContactsModalOpen(false)}
        onSelectContact={handleSelectContact}
        onRemoveContact={handleRemoveContact}
      />

      {/* Settings Modal */}
      <SettingsModal isOpen={settingsModalOpen} onClose={() => setSettingsModalOpen(false)} />

      {/* Backdrop */}
      <div
        className={`slide-menu-backdrop ${isOpen ? 'open' : ''}`}
        onClick={handleBackdropClick}
        aria-hidden="true"
      />

      {/* Menu Panel */}
      <div
        className={`slide-menu ${isOpen ? 'open' : ''}`}
        role="dialog"
        aria-modal="true"
        aria-label="Main menu"
      >
        {/* User header */}
        <div className="slide-menu__header">
          <span className="avatar avatar-lg">
            <img
              src={user?.avatar_url || '/img/default-avatar.svg'}
              alt="avatar"
            />
          </span>
          <div className="slide-menu__user-info">
            <span className="slide-menu__user-name">
              {user?.name || user?.username}
            </span>
            <span className="slide-menu__user-username">@{user?.username}</span>
          </div>
        </div>

        {/* Menu Items */}
        <nav className="slide-menu__nav">
          <button
            className="slide-menu__item"
            onClick={handleOpenProfile}
          >
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                <circle cx="12" cy="7" r="4" />
              </svg>
            </span>
            <span>Мой профиль</span>
          </button>

          <button
            className={`slide-menu__item ${isActive('/') ? 'active' : ''}`}
            onClick={() => handleNavigate('/')}
          >
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
              </svg>
            </span>
            <span>Чаты</span>
          </button>

          <button className="slide-menu__item" disabled>
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </span>
            <span>Новый групповой чат</span>
            <span className="slide-menu__badge">Скоро</span>
          </button>

          <button
            className="slide-menu__item"
            onClick={handleOpenContacts}
          >
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <line x1="19" y1="8" x2="19" y2="14" />
                <line x1="22" y1="11" x2="16" y2="11" />
              </svg>
            </span>
            <span>Контакты</span>
          </button>

          <div className="slide-menu__divider" />

          <button
            className="slide-menu__item"
            onClick={handleOpenSettings}
          >
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="3" />
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
              </svg>
            </span>
            <span>Настройки</span>
          </button>

          <div className="slide-menu__theme-switch">
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
              </svg>
            </span>
            <span>Тёмная тема</span>
            <label className="toggle-switch">
              <input
                type="checkbox"
                checked={theme === 'dark'}
                onChange={toggleTheme}
              />
              <span className="toggle-slider" />
            </label>
          </div>

          <div className="slide-menu__divider" />

          <button
            className="slide-menu__item slide-menu__item--danger"
            onClick={handleLogout}
          >
            <span className="slide-menu__icon">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                <polyline points="16 17 21 12 16 7" />
                <line x1="21" y1="12" x2="9" y2="12" />
              </svg>
            </span>
            <span>Выйти</span>
          </button>
        </nav>
      </div>
    </>
  )
}
