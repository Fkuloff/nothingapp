import { useState, useEffect } from 'react'
import { searchUsers } from '../../shared/api/contactsApi'
import type { UserListItem } from '../../shared/api/types'

type Props = {
  onAddContact: (userId: number) => Promise<void>
  existingContactIds?: Set<number>
}

export function UserSearchBox({ onAddContact, existingContactIds = new Set() }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<UserListItem[]>([])
  const [searching, setSearching] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [addingUserId, setAddingUserId] = useState<number | null>(null)

  // Debounced search effect
  useEffect(() => {
    if (query.length < 2) {
      setResults([])
      setError(null)
      return
    }

    const timeout = setTimeout(async () => {
      setSearching(true)
      setError(null)

      try {
        const users = await searchUsers(query)
        setResults(users)
      } catch (err) {
        setError('Не удалось найти пользователей')
        console.error('Search failed:', err)
      } finally {
        setSearching(false)
      }
    }, 300)

    return () => clearTimeout(timeout)
  }, [query])

  const handleAddContact = async (userId: number) => {
    setAddingUserId(userId)
    try {
      await onAddContact(userId)
      // Remove from search results after adding
      setResults((prev) => prev.filter((user) => user.id !== userId))
    } catch (err) {
      // Error is handled by parent component via toast
      console.error('Failed to add contact:', err)
    } finally {
      setAddingUserId(null)
    }
  }

  return (
    <div className="chat-list__new">
      <label className="chat-list__label" htmlFor="userSearch">
        Найти новых контактов
      </label>
      <input
        id="userSearch"
        type="search"
        className="form-control"
        placeholder="Поиск по имени или username..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />

      {searching && <p className="text-muted small mt-2">Поиск...</p>}

      {error && <p className="text-danger small mt-2">{error}</p>}

      {!searching && query.length >= 2 && results.length === 0 && !error && (
        <p className="text-muted small mt-2">Пользователи не найдены</p>
      )}

      {results.length > 0 && (
        <div className="mt-3">
          <ul className="list-unstyled">
            {results.map((user) => {
              const isExisting = existingContactIds.has(user.id)
              const isAdding = addingUserId === user.id

              return (
                <li key={user.id} className="chat-list-item mb-2">
                  <span className="avatar avatar-sm">
                    <img src={user.avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
                  </span>
                  <div className="flex-grow-1">
                    <div className="chat-list-item__name">{user.name}</div>
                    <div className="text-muted small">@{user.username}</div>
                  </div>
                  <button
                    className="btn btn-sm btn-primary"
                    onClick={() => handleAddContact(user.id)}
                    disabled={isExisting || isAdding}
                  >
                    {isAdding ? 'Добавляем...' : isExisting ? 'Уже в контактах' : 'Добавить'}
                  </button>
                </li>
              )
            })}
          </ul>
        </div>
      )}
    </div>
  )
}
