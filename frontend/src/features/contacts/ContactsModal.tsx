import { useEffect, useState } from 'react'

import { addContact, getContacts, searchUsers } from '../../shared/api/contactsApi'
import { resolveApiUrl } from '../../shared/api/httpClient'
import type { UserListItem } from '../../shared/api/types'
import { ChatBubbleIcon, CloseIcon, PersonAddIcon } from '../../shared/components/Icons'
import { useAndroidBack } from '../../shared/hooks/useAndroidBack'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  isOpen: boolean
  onClose: () => void
  onSelectContact: (contact: UserListItem) => void | Promise<void>
  onRemoveContact?: (userId: number) => void | Promise<void>
}

export function ContactsModal({ isOpen, onClose, onSelectContact, onRemoveContact }: Props) {
  const [contacts, setContacts] = useState<UserListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [searchResults, setSearchResults] = useState<UserListItem[]>([])
  const [searching, setSearching] = useState(false)
  const [startingId, setStartingId] = useState<number | null>(null)
  const [confirmingId, setConfirmingId] = useState<number | null>(null)
  const [removingId, setRemovingId] = useState<number | null>(null)
  const [addingId, setAddingId] = useState<number | null>(null)
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })
  useAndroidBack(() => { onClose(); return true }, isOpen)

  // Auto-reset confirmation after 5 seconds
  useEffect(() => {
    if (confirmingId === null) return
    const timer = setTimeout(() => setConfirmingId(null), 5000)
    return () => clearTimeout(timer)
  }, [confirmingId])

  // Load contacts on open
  useEffect(() => {
    if (!isOpen) return

    let cancelled = false
    setLoading(true)
    getContacts()
      .then((data) => {
        if (!cancelled) setContacts(data)
      })
      .catch(console.error)
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [isOpen])

  // Reset state on close
  useEffect(() => {
    if (!isOpen) {
      setSearch('')
      setSearchResults([])
      setSearching(false)
      setConfirmingId(null)
      setRemovingId(null)
    }
  }, [isOpen])

  // Debounced global search (when query >= 2 chars)
  useEffect(() => {
    if (search.length < 2) {
      setSearchResults([])
      return
    }

    const timeout = setTimeout(async () => {
      setSearching(true)
      try {
        const users = await searchUsers(search)
        setSearchResults(users)
      } catch (err) {
        console.error('Search failed:', err)
      } finally {
        setSearching(false)
      }
    }, 300)

    return () => clearTimeout(timeout)
  }, [search])

  const handleContactClick = async (contact: UserListItem) => {
    setStartingId(contact.id)
    try {
      await onSelectContact(contact)
      onClose()
    } finally {
      setStartingId(null)
    }
  }

  const handleConfirmRemove = async (userId: number) => {
    if (!onRemoveContact) return
    setRemovingId(userId)
    try {
      await onRemoveContact(userId)
      // Remove from local state
      setContacts((prev) => prev.filter((c) => c.id !== userId))
      setConfirmingId(null)
    } finally {
      setRemovingId(null)
    }
  }

  const handleAddContact = async (user: UserListItem) => {
    setAddingId(user.id)
    try {
      await addContact(user.id)
      setContacts((prev) => [...prev, user])
    } catch (err) {
      console.error('Failed to add contact:', err)
    } finally {
      setAddingId(null)
    }
  }

  // Filter existing contacts locally
  const filteredContacts = contacts.filter(
    (c) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.username.toLowerCase().includes(search.toLowerCase())
  )

  // Merge: show search results that are NOT in contacts
  const contactIds = new Set(contacts.map((c) => c.id))
  const globalResults = searchResults.filter((u) => !contactIds.has(u.id))

  const hasGlobalSearch = search.length >= 2
  const showGlobalResults = hasGlobalSearch && globalResults.length > 0

  if (!isOpen) return null

  const renderContactItem = (user: UserListItem, index: number, isContact: boolean) => {
    const isLoading = startingId === user.id
    const isConfirming = confirmingId === user.id
    const isRemoving = removingId === user.id

    return (
      <div
        key={user.id}
        className={`contacts-modal__item${isConfirming ? ' contacts-modal__item--confirming' : ''}`}
        style={{ animationDelay: `${index * 30}ms` }}
        role="listitem"
      >
        <div className="contacts-modal__avatar-wrap">
          <img
            src={resolveApiUrl(user.avatar_url) || '/img/default-avatar.svg'}
            alt=""
            className="contacts-modal__avatar"
          />
        </div>
        <div className="contacts-modal__info">
          {isConfirming ? (
            <span className="contact-card__confirm-text">Удалить {user.name}?</span>
          ) : (
            <>
              <span className="contacts-modal__name">{user.name}</span>
              <span className="contacts-modal__username">@{user.username}</span>
            </>
          )}
        </div>

        {isConfirming ? (
          <div className="contact-card__confirm-actions">
            <button
              className="contact-card__confirm-btn contact-card__confirm-btn--delete"
              onClick={() => handleConfirmRemove(user.id)}
              disabled={isRemoving}
            >
              {isRemoving ? <span className="contacts-modal__spinner" /> : 'Удалить'}
            </button>
            <button
              className="contact-card__confirm-btn contact-card__confirm-btn--cancel"
              onClick={() => setConfirmingId(null)}
            >
              Отмена
            </button>
          </div>
        ) : (
          <div className="contacts-modal__actions-group">
            <button
              className="contacts-modal__action-btn"
              onClick={() => handleContactClick(user)}
              disabled={isLoading}
              tabIndex={0}
            >
              {isLoading ? (
                <span className="contacts-modal__spinner" />
              ) : (
                <>
                  <ChatBubbleIcon />
                  <span>Чат</span>
                </>
              )}
            </button>
            {!isContact && (
              <button
                className="contacts-modal__action-btn contacts-modal__action-btn--add"
                onClick={(e) => { e.stopPropagation(); handleAddContact(user) }}
                disabled={addingId === user.id}
                tabIndex={0}
                title="Добавить в контакты"
              >
                {addingId === user.id ? (
                  <span className="contacts-modal__spinner" />
                ) : (
                  <PersonAddIcon size={22} />
                )}
              </button>
            )}
            {isContact && onRemoveContact && (
              <button
                className="contacts-modal__action-btn contacts-modal__action-btn--danger"
                onClick={() => setConfirmingId(user.id)}
                tabIndex={0}
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="contacts-modal-backdrop" onClick={handleBackdropClick}>
      <div className="contacts-modal" role="dialog" aria-modal="true">
        <div className="contacts-modal__header">
          <h2 className="contacts-modal__title">Контакты</h2>
          <button className="contacts-modal__close" onClick={onClose} aria-label="Закрыть">
            <CloseIcon />
          </button>
        </div>

        <div className="contacts-modal__search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            type="text"
            placeholder="Поиск контактов и пользователей..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
          />
        </div>

        <div className="contacts-modal__list" role="list">
          {loading ? (
            <div className="contacts-modal__empty">
              <span className="contacts-modal__spinner" style={{ width: 20, height: 20, borderWidth: 2, display: 'inline-block', verticalAlign: 'middle' }} />
            </div>
          ) : (
            <>
              {/* Existing contacts */}
              {filteredContacts.length > 0 && (
                <>
                  {showGlobalResults && (
                    <div className="contacts-modal__section-label">Контакты</div>
                  )}
                  {filteredContacts.map((contact, i) => renderContactItem(contact, i, true))}
                </>
              )}

              {/* Global search results (not in contacts) */}
              {showGlobalResults && (
                <>
                  <div className="contacts-modal__section-label">Найдено</div>
                  {globalResults.map((user, i) => renderContactItem(user, filteredContacts.length + i, false))}
                </>
              )}

              {/* Searching indicator */}
              {searching && (
                <div className="contacts-modal__empty">
                  <span className="contacts-modal__spinner" style={{ width: 20, height: 20, borderWidth: 2, display: 'inline-block', verticalAlign: 'middle' }} />
                </div>
              )}

              {/* Empty states */}
              {filteredContacts.length === 0 && !showGlobalResults && !searching && (
                <div className="contacts-modal__empty">
                  {search.length >= 2
                    ? 'Никого не найдено'
                    : search
                      ? 'Введите минимум 2 символа для поиска'
                      : 'У вас пока нет контактов'}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
