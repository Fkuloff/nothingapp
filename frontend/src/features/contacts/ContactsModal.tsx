import { useEffect, useState } from 'react'

import { addContact, getContacts, searchUsers } from '../../shared/api/contactsApi'
import { resolveApiUrl } from '../../shared/api/httpClient'
import type { UserListItem } from '../../shared/api/types'
import { ChatBubbleIcon, CloseIcon, PersonAddIcon } from '../../shared/components/Icons'
import { useAndroidBack } from '../../shared/hooks/useAndroidBack'
import { useLongPress } from '../../shared/hooks/useLongPress'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  isOpen: boolean
  onClose: () => void
  onSelectContact: (contact: UserListItem) => void | Promise<void>
  onRemoveContact?: (userId: number) => void | Promise<void>
}

type RowProps = {
  user: UserListItem
  index: number
  isContact: boolean
  isLoading: boolean
  isConfirming: boolean
  isRemoving: boolean
  isAdding: boolean
  canRemove: boolean
  onStartChat: (user: UserListItem) => void
  onAddContact: (user: UserListItem) => void
  onRequestRemove: (userId: number) => void
  onConfirmRemove: (userId: number) => void
  onCancelRemove: () => void
}

function ContactRow({
  user,
  index,
  isContact,
  isLoading,
  isConfirming,
  isRemoving,
  isAdding,
  canRemove,
  onStartChat,
  onAddContact,
  onRequestRemove,
  onConfirmRemove,
  onCancelRemove,
}: RowProps) {
  const longPress = useLongPress(() => {
    if (canRemove && !isConfirming) onRequestRemove(user.id)
  })

  const handleRowClick = () => {
    if (isConfirming || isLoading) return
    onStartChat(user)
  }

  return (
    <div
      className={`contacts-modal__item${isConfirming ? ' contacts-modal__item--confirming' : ''}`}
      style={{ animationDelay: `${index * 30}ms` }}
      role="listitem"
      onClick={handleRowClick}
      {...longPress}
    >
      <div className="contacts-modal__avatar-wrap">
        <img
          src={resolveApiUrl(user.avatar_url) || '/img/default-avatar.svg'}
          alt=""
          className="contacts-modal__avatar"
          loading="lazy"
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
        <div className="contact-card__confirm-actions" onClick={(e) => e.stopPropagation()}>
          <button
            className="contact-card__confirm-btn contact-card__confirm-btn--delete"
            onClick={() => onConfirmRemove(user.id)}
            disabled={isRemoving}
          >
            {isRemoving ? <span className="contacts-modal__spinner" /> : 'Удалить'}
          </button>
          <button
            className="contact-card__confirm-btn contact-card__confirm-btn--cancel"
            onClick={onCancelRemove}
          >
            Отмена
          </button>
        </div>
      ) : (
        <div className="contacts-modal__actions-group" onClick={(e) => e.stopPropagation()}>
          {isLoading && <span className="contacts-modal__spinner" />}
          <button
            className="contacts-modal__action-btn contacts-modal__action-btn--chat-hint"
            onClick={() => onStartChat(user)}
            disabled={isLoading}
            tabIndex={0}
            aria-label="Открыть чат"
          >
            <ChatBubbleIcon />
            <span>Чат</span>
          </button>
          {!isContact && (
            <button
              className="contacts-modal__action-btn contacts-modal__action-btn--add"
              onClick={() => onAddContact(user)}
              disabled={isAdding}
              tabIndex={0}
              title="Добавить в контакты"
              aria-label="Добавить в контакты"
            >
              {isAdding ? <span className="contacts-modal__spinner" /> : <PersonAddIcon size={22} />}
            </button>
          )}
          {canRemove && (
            <button
              className="contacts-modal__action-btn contacts-modal__action-btn--danger contacts-modal__action-btn--desktop-only"
              onClick={() => onRequestRemove(user.id)}
              tabIndex={0}
              aria-label="Удалить контакт"
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

  useEffect(() => {
    if (confirmingId === null) return
    const timer = setTimeout(() => setConfirmingId(null), 5000)
    return () => clearTimeout(timer)
  }, [confirmingId])

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

  useEffect(() => {
    if (!isOpen) {
      setSearch('')
      setSearchResults([])
      setSearching(false)
      setConfirmingId(null)
      setRemovingId(null)
    }
  }, [isOpen])

  useEffect(() => {
    // Strip the leading "@" before talking to the backend — users naturally copy
    // "@username" out of the UI, but the username column doesn't contain the @ sigil.
    // Doing this on the client means the search works even on backends that haven't
    // been redeployed with the server-side strip.
    const query = search.replace(/^@/, '').trim()
    if (query.length < 2) {
      setSearchResults([])
      return
    }

    const timeout = setTimeout(async () => {
      setSearching(true)
      try {
        const users = await searchUsers(query)
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

  // Normalize the query the same way the backend does: strip leading @ and split into
  // whitespace-separated tokens. Each token must appear (substring match) in either
  // username or name. Token order doesn't matter, so "Ivanov Ivan" matches "Ivan Ivanov".
  const normalizedTokens = search
    .replace(/^@/, '')
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)

  const filteredContacts = normalizedTokens.length === 0
    ? contacts
    : contacts.filter((c) => {
        const name = c.name.toLowerCase()
        const username = c.username.toLowerCase()
        return normalizedTokens.every((t) => name.includes(t) || username.includes(t))
      })

  const contactIds = new Set(contacts.map((c) => c.id))
  const globalResults = searchResults.filter((u) => !contactIds.has(u.id))

  // The "real" query length (after stripping the leading @) is what determines whether
  // the global lookup ran — keep the empty-state hint and the global-results gate consistent
  // with it.
  const queryLengthForSearch = search.replace(/^@/, '').trim().length
  const hasGlobalSearch = queryLengthForSearch >= 2
  const showGlobalResults = hasGlobalSearch && globalResults.length > 0

  if (!isOpen) return null

  const renderRow = (user: UserListItem, index: number, isContact: boolean) => (
    <ContactRow
      key={user.id}
      user={user}
      index={index}
      isContact={isContact}
      isLoading={startingId === user.id}
      isConfirming={confirmingId === user.id}
      isRemoving={removingId === user.id}
      isAdding={addingId === user.id}
      canRemove={isContact && Boolean(onRemoveContact)}
      onStartChat={handleContactClick}
      onAddContact={handleAddContact}
      onRequestRemove={(id) => setConfirmingId(id)}
      onConfirmRemove={handleConfirmRemove}
      onCancelRemove={() => setConfirmingId(null)}
    />
  )

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
              {filteredContacts.length > 0 && (
                <>
                  {showGlobalResults && (
                    <div className="contacts-modal__section-label">Контакты</div>
                  )}
                  {filteredContacts.map((contact, i) => renderRow(contact, i, true))}
                </>
              )}

              {showGlobalResults && (
                <>
                  <div className="contacts-modal__section-label">Найдено</div>
                  {globalResults.map((user, i) => renderRow(user, filteredContacts.length + i, false))}
                </>
              )}

              {searching && (
                <div className="contacts-modal__empty">
                  <span className="contacts-modal__spinner" style={{ width: 20, height: 20, borderWidth: 2, display: 'inline-block', verticalAlign: 'middle' }} />
                </div>
              )}

              {filteredContacts.length === 0 && !showGlobalResults && !searching && (
                <div className="contacts-modal__empty">
                  {queryLengthForSearch >= 2
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
