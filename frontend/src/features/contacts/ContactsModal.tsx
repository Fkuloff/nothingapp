import { useEffect, useState } from 'react'
import { getContacts } from '../../shared/api/contactsApi'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'
import type { UserListItem } from '../../shared/api/types'

type Props = {
  isOpen: boolean
  onClose: () => void
  onSelectContact: (contact: UserListItem) => void
}

export function ContactsModal({ isOpen, onClose, onSelectContact }: Props) {
  const [contacts, setContacts] = useState<UserListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const { handleBackdropClick } = useModalBehavior({ isOpen, onClose })

  useEffect(() => {
    if (!isOpen) return

    let cancelled = false
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

  const handleContactClick = (contact: UserListItem) => {
    onSelectContact(contact)
    onClose()
  }

  const filteredContacts = contacts.filter(
    (c) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.username.toLowerCase().includes(search.toLowerCase())
  )

  if (!isOpen) return null

  return (
    <div className="contacts-modal-backdrop" onClick={handleBackdropClick}>
      <div className="contacts-modal" role="dialog" aria-modal="true">
        <div className="contacts-modal__header">
          <h2 className="contacts-modal__title">Контакты</h2>
          <button className="contacts-modal__close" onClick={onClose} aria-label="Закрыть">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="contacts-modal__search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            type="text"
            placeholder="Поиск контактов..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
          />
        </div>

        <div className="contacts-modal__list">
          {loading ? (
            <div className="contacts-modal__empty">Загрузка...</div>
          ) : filteredContacts.length === 0 ? (
            <div className="contacts-modal__empty">
              {search ? 'Контакты не найдены' : 'У вас пока нет контактов'}
            </div>
          ) : (
            filteredContacts.map((contact) => (
              <button
                key={contact.id}
                className="contacts-modal__item"
                onClick={() => handleContactClick(contact)}
              >
                <img
                  src={contact.avatar_url || '/img/default-avatar.svg'}
                  alt=""
                  className="contacts-modal__avatar"
                />
                <div className="contacts-modal__info">
                  <span className="contacts-modal__name">{contact.name}</span>
                  <span className="contacts-modal__username">@{contact.username}</span>
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
