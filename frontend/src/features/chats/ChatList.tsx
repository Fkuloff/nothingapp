import { useMemo, useState } from 'react'
import type { ChatItem } from '../../shared/api/types'
import { createChatByUsername } from '../../shared/api/chatsApi'

type Props = {
  chats: ChatItem[]
  activeChatId?: number
  onSelect: (chatId: number) => void
  onChatCreated?: () => void
  loading?: boolean
  error?: string | null
}

export function ChatList({ chats, activeChatId, onSelect, onChatCreated, loading, error }: Props) {
  const [newChatUsername, setNewChatUsername] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const sortedChats = useMemo(
    () => [...chats].sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
    [chats]
  )

  const handleCreateChat = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!newChatUsername.trim()) return

    setCreating(true)
    setCreateError(null)

    try {
      await createChatByUsername(newChatUsername.trim())
      setNewChatUsername('')
      onChatCreated?.()
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Не удалось создать чат')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="telegram-chat-list">
      {/* Compact new chat form */}
      <div className="telegram-chat-list__new">
        <form onSubmit={handleCreateChat} className="telegram-new-chat-form">
          <input
            type="text"
            className="form-control"
            name="other_username"
            placeholder="Новый чат: @username"
            value={newChatUsername}
            onChange={(e) => setNewChatUsername(e.target.value)}
            disabled={creating}
          />
          <button
            type="submit"
            className="btn btn-primary"
            disabled={creating || !newChatUsername.trim()}
            title="Создать чат"
          >
            +
          </button>
        </form>
        {createError && <p className="text-danger small mb-0 mt-1">{createError}</p>}
      </div>

      {error && <div className="alert alert-danger alert-sm m-2">{error}</div>}

      {/* Chat list items */}
      <div className="telegram-chat-list__items fancy-scroll">
        {loading ? (
          <p className="text-muted small p-3">Загружаем диалоги...</p>
        ) : sortedChats.length === 0 ? (
          <div className="telegram-empty-list">
            <p>Пока пусто</p>
            <p className="text-muted small">Создайте новый чат выше</p>
          </div>
        ) : (
          <ul className="telegram-chat-ul">
            {sortedChats.map((chat) => {
              const isActive = chat.id === activeChatId
              const hasUnread = chat.unread_count > 0

              return (
                <li
                  key={chat.id}
                  className={`chat-list-item${isActive ? ' active' : ''}${hasUnread ? ' has-unread' : ''}`}
                  role="button"
                  onClick={() => onSelect(chat.id)}
                  tabIndex={0}
                >
                  <span className="avatar avatar-md">
                    <img src={chat.avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
                  </span>
                  <div className="chat-list-item-content">
                    <div className="chat-list-item__top">
                      <span className="chat-list-item__name">{chat.other_user_name}</span>
                      <span className="chat-list-item__time">
                        {new Date(chat.updated_at).toLocaleTimeString('ru-RU', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                    <div className="chat-list-item__preview">
                      <span className="text-muted text-truncate">
                        {chat.last_message || 'Нет сообщений'}
                      </span>
                      {hasUnread && (
                        <span className="unread-badge">{chat.unread_count}</span>
                      )}
                    </div>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
