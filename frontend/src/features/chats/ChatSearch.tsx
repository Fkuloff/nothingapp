import { useEffect, useMemo, useRef, useState } from 'react'

import type { Message } from '../../shared/api/types'

type Props = {
  messages: Message[]
  onResultClick: (messageId: number) => void
  onClose: () => void
}

export function ChatSearch({ messages, onResultClick, onClose }: Props) {
  const [query, setQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  // Search the already-decrypted messages held by ChatWindow. The previous
  // version re-fetched via getChatMessages and searched msg.text — but that's
  // the scheme=2 ciphertext, so it never matched anything the user typed.
  // Case-insensitive (incl. Cyrillic) via toLowerCase on both sides.
  const results = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return []
    return messages.filter((m) => !m.is_deleted && m.text && m.text.toLowerCase().includes(q))
  }, [messages, query])

  const highlightMatch = (text: string, q: string) => {
    if (!q.trim()) return text
    const parts = text.split(new RegExp(`(${q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'))
    return parts.map((part, i) =>
      part.toLowerCase() === q.toLowerCase() ? <mark key={i}>{part}</mark> : part,
    )
  }

  const truncateText = (text: string, maxLen = 100) =>
    text.length <= maxLen ? text : text.slice(0, maxLen) + '...'

  return (
    <div className="chat-search">
      <div className="chat-search__header">
        <input
          ref={inputRef}
          type="search"
          className="form-control"
          placeholder="Поиск по сообщениям..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button className="chat-search__close" onClick={onClose} aria-label="Закрыть поиск">
          &times;
        </button>
      </div>

      {query.trim() && results.length === 0 && (
        <div className="chat-search__status">Ничего не найдено</div>
      )}

      {results.length > 0 && (
        <div className="chat-search__results">
          <div className="chat-search__count">Найдено: {results.length}</div>
          {results.map((result, index) => (
            <button
              key={`${result.id}-${index}`}
              className="chat-search__result"
              onClick={() => onResultClick(result.id)}
            >
              <div className="chat-search__result-text">
                {highlightMatch(truncateText(result.text), query)}
              </div>
              <div className="chat-search__result-date">
                {new Date(result.created_at).toLocaleString()}
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
