import { useCallback, useEffect, useMemo,useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'

import type { OutletContextType } from '../App'
import { ContactItem } from '../features/contacts/ContactItem'
import { UserSearchBox } from '../features/contacts/UserSearchBox'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import { createChat } from '../shared/api/chatsApi'
import { addContact, getContacts, removeContact } from '../shared/api/contactsApi'
import type { UserListItem } from '../shared/api/types'
import { useToast } from '../shared/components/ToastContext'

export default function ContactsPage() {
  const { setMenuOpen } = useOutletContext<OutletContextType>()
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [contacts, setContacts] = useState<UserListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Load contacts on mount
  const loadContacts = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await getContacts()
      setContacts(data)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Не удалось загрузить контакты'
      setError(errorMessage)
      showToast(errorMessage, 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  useEffect(() => {
    loadContacts()
  }, [loadContacts])

  // Start chat with contact (create or open existing chat)
  const handleStartChat = useCallback(
    async (userId: number) => {
      try {
        const chat = await createChat(userId)
        navigate(`/?chat=${chat.id}`)
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Не удалось создать чат'
        showToast(errorMessage, 'error')
      }
    },
    [navigate, showToast]
  )

  // Remove contact
  const handleRemoveContact = useCallback(
    async (userId: number) => {
      try {
        await removeContact(userId)
        showToast('Контакт удалён', 'success')
        // Update local state immediately
        setContacts((prev) => prev.filter((contact) => contact.id !== userId))
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Не удалось удалить контакт'
        showToast(errorMessage, 'error')
      }
    },
    [showToast]
  )

  // Add new contact from search
  const handleAddContact = useCallback(
    async (userId: number) => {
      try {
        await addContact(userId)
        showToast('Контакт добавлен', 'success')
        // Reload contacts to get the new contact with full data
        await loadContacts()
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Не удалось добавить контакт'
        showToast(errorMessage, 'error')
        throw err // Re-throw so UserSearchBox can handle the error state
      }
    },
    [showToast, loadContacts]
  )

  // Create a Set of existing contact IDs for efficient lookup
  const existingContactIds = useMemo(() => {
    return new Set(contacts.map((contact) => contact.id))
  }, [contacts])

  return (
    <div className="page-container">
      {/* Header */}
      <div className="page-header">
        <HamburgerButton onClick={() => setMenuOpen(true)} />
        <h2>Контакты</h2>
        <span className="chip">{contacts.length}</span>
      </div>

      {/* Content */}
      <div className="page-content">
        {/* Search box */}
        <UserSearchBox onAddContact={handleAddContact} onStartChat={handleStartChat} existingContactIds={existingContactIds} />

        {/* Error state */}
        {error && !loading && <div className="alert alert-danger alert-sm mt-3">{error}</div>}

        {/* Contacts list */}
        <div className="mt-3">
          {loading ? (
            <div className="contacts-modal__empty">
              <span className="contacts-modal__spinner" style={{ width: 22, height: 22, borderWidth: 2, display: 'inline-block' }} />
            </div>
          ) : contacts.length === 0 ? (
            <div className="contacts-page-empty">
              <svg className="contacts-page-empty__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <line x1="19" y1="8" x2="19" y2="14" />
                <line x1="22" y1="11" x2="16" y2="11" />
              </svg>
              <p className="contacts-page-empty__title">Контактов пока нет</p>
              <p className="contacts-page-empty__hint">
                Используйте поиск выше, чтобы найти пользователей
              </p>
            </div>
          ) : (
            <ul className="contacts-page-list">
              {contacts.map((contact) => (
                <ContactItem
                  key={contact.id}
                  id={contact.id}
                  username={contact.username}
                  name={contact.name}
                  avatar_url={contact.avatar_url}
                  onStartChat={handleStartChat}
                  onRemove={handleRemoveContact}
                />
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
