import { useCallback, useEffect,useRef, useState } from 'react'

import { getChatMessages } from '../../shared/api/chatsApi'

type SearchResult = {
  messageId: number
  text: string
  createdAt: string
  matchType: 'text'
}

type Props = {
  chatId: number
  onResultClick: (messageId: number) => void
  onClose: () => void
}

export function ChatSearch({ chatId, onResultClick, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const debounceRef = useRef<number | undefined>(undefined)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const performSearch = useCallback(
    async (searchQuery: string) => {
      if (!searchQuery.trim()) {
        setResults([])
        return
      }

      setSearching(true)

      try {
        // Load all messages for this chat
        const messages = await getChatMessages(chatId)
        const q = searchQuery.toLowerCase()
        const found: SearchResult[] = []

        for (const msg of messages) {
          if (msg.is_deleted) continue

          // Search in message text. Filename search was removed when
          // attachment metadata became client-side encrypted — the filename
          // lives inside encrypted_metadata under file_key, and decrypting
          // every attachment in the chat on every keystroke would be
          // expensive (and we'd need to cache decrypted names somewhere).
          // Re-introduce when there's a real demand.
          if (msg.text.toLowerCase().includes(q)) {
            found.push({
              messageId: msg.id,
              text: msg.text,
              createdAt: msg.created_at,
              matchType: 'text',
            })
          }
        }

        setResults(found)
      } catch (err) {
        console.error('Search failed:', err)
        setResults([])
      } finally {
        setSearching(false)
      }
    },
    [chatId],
  )

  const handleQueryChange = useCallback(
    (value: string) => {
      setQuery(value)

      if (debounceRef.current) {
        window.clearTimeout(debounceRef.current)
      }

      debounceRef.current = window.setTimeout(() => {
        performSearch(value)
      }, 300)
    },
    [performSearch],
  )

  const highlightMatch = (text: string, query: string) => {
    if (!query.trim()) return text

    const parts = text.split(new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'))
    return parts.map((part, i) =>
      part.toLowerCase() === query.toLowerCase() ? (
        <mark key={i}>{part}</mark>
      ) : (
        part
      ),
    )
  }

  const truncateText = (text: string, maxLen = 100) => {
    if (text.length <= maxLen) return text
    return text.slice(0, maxLen) + '...'
  }

  return (
    <div className="chat-search">
      <div className="chat-search__header">
        <input
          ref={inputRef}
          type="search"
          className="form-control"
          placeholder="Поиск по сообщениям..."
          value={query}
          onChange={(e) => handleQueryChange(e.target.value)}
        />
        <button className="chat-search__close" onClick={onClose} aria-label="Закрыть поиск">
          &times;
        </button>
      </div>

      {searching && (
        <div className="chat-search__status">
          Поиск...
        </div>
      )}

      {!searching && query && results.length === 0 && (
        <div className="chat-search__status">Ничего не найдено</div>
      )}

      {results.length > 0 && (
        <div className="chat-search__results">
          <div className="chat-search__count">
            Найдено: {results.length}
          </div>
          {results.map((result, index) => (
            <button
              key={`${result.messageId}-${index}`}
              className="chat-search__result"
              onClick={() => onResultClick(result.messageId)}
            >
              <div className="chat-search__result-text">
                {highlightMatch(truncateText(result.text), query)}
              </div>
              <div className="chat-search__result-date">
                {new Date(result.createdAt).toLocaleString()}
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
