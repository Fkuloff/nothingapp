import { useEffect, useState, useCallback, useMemo } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { getContacts, addContact, removeContact } from '../shared/api/contactsApi'
import type { UserListItem } from '../shared/api/types'
import { ContactItem } from '../features/contacts/ContactItem'
import { UserSearchBox } from '../features/contacts/UserSearchBox'
import { useToast } from '../shared/components/ToastContext'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import type { OutletContextType } from '../App'

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
        <UserSearchBox onAddContact={handleAddContact} existingContactIds={existingContactIds} />

        {/* Error state */}
        {error && !loading && <div className="alert alert-danger alert-sm mt-3">{error}</div>}

        {/* Contacts list */}
        <div className="mt-3">
          {loading ? (
            <p className="text-muted small">Загружаем контакты...</p>
          ) : contacts.length === 0 ? (
            <div className="telegram-empty-list">
              <p>Контактов пока нет</p>
              <p className="text-muted small">
                Используйте поиск выше, чтобы найти пользователей
              </p>
            </div>
          ) : (
            <ul className="list-unstyled mb-0">
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
