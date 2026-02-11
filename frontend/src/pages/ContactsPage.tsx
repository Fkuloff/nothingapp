import { useEffect, useState, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { getContacts, addContact, removeContact } from '../shared/api/contactsApi'
import type { UserListItem } from '../shared/api/types'
import { ContactItem } from '../features/contacts/ContactItem'
import { UserSearchBox } from '../features/contacts/UserSearchBox'
import { useToast } from '../shared/components/ToastContext'

export default function ContactsPage() {
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

  // Start chat with contact (navigate to their profile)
  const handleStartChat = useCallback(
    (userId: number) => {
      navigate(`/profile/${userId}`)
    },
    [navigate]
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
    <div className="workspace">
      <div className="workspace__panel" style={{ gridColumn: '1 / -1', maxWidth: '800px', margin: '0 auto' }}>
        <div className="chat-list">
          {/* Header */}
          <div className="chat-list__header">
            <div>
              <p className="eyebrow">Контакты</p>
              <h3>Ваши контакты</h3>
            </div>
            <div className="chat-list__badges">
              <span className="chat-list__badge">{contacts.length}</span>
            </div>
          </div>

          {/* Search box */}
          <UserSearchBox onAddContact={handleAddContact} existingContactIds={existingContactIds} />

          {/* Error state */}
          {error && !loading && <div className="alert alert-danger alert-sm mt-3">{error}</div>}

          {/* Contacts list */}
          <div className="chat-list__items">
            {loading ? (
              <p className="text-muted small">Загружаем контакты...</p>
            ) : contacts.length === 0 ? (
              <div className="empty-list">
                <p className="empty-list__title">Контактов пока нет</p>
                <p className="empty-list__text">
                  Используйте поиск выше, чтобы найти пользователей и добавить их в контакты.
                </p>
              </div>
            ) : (
              <ul className="list-unstyled mb-0 chat-list__ul">
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
    </div>
  )
}
