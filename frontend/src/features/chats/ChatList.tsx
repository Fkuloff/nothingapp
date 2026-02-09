import { useMemo, useState } from 'react'
import type { ChatItem } from '../../shared/api/types'
import { createChat } from '../../shared/api/chatsApi'

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
      await createChat(newChatUsername.trim())
      setNewChatUsername('')
      onChatCreated?.()
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Не удалось создать чат')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="chat-list">
      <div className="chat-list__header">
        <div>
          <p className="eyebrow">Диалоги</p>
          <h3>Ваши чаты</h3>
        </div>
        <div className="chat-list__badge">{chats.length}</div>
      </div>

      <div className="chat-list__new">
        <form onSubmit={handleCreateChat} className="chat-list__form">
          <label className="chat-list__label" htmlFor="newChat">
            Новый чат по username
          </label>
          <div className="input-group">
            <input
              id="newChat"
              type="text"
              className="form-control"
              name="other_username"
              placeholder="@username"
              value={newChatUsername}
              onChange={(e) => setNewChatUsername(e.target.value)}
              disabled={creating}
              required
            />
            <button type="submit" className="btn btn-primary" disabled={creating}>
              {creating ? 'Создаём...' : 'Создать'}
            </button>
          </div>
          {createError && <p className="text-danger small mb-0 mt-1">{createError}</p>}
        </form>
      </div>

      {error && <div className="alert alert-danger alert-sm">{error}</div>}

      <div className="chat-list__items">
        {loading ? (
          <p className="text-muted small">Загружаем диалоги...</p>
        ) : sortedChats.length === 0 ? (
          <div className="empty-list">
            <p className="empty-list__title">Пока пусто</p>
            <p className="empty-list__text">Создайте новый чат, чтобы начать разговор.</p>
          </div>
        ) : (
          <ul className="list-unstyled mb-0 chat-list__ul">
            {sortedChats.map((chat) => {
              const isActive = chat.id === activeChatId

              return (
                <li
                  key={chat.id}
                  className={`chat-list-item${isActive ? ' active' : ''}`}
                  role="button"
                  onClick={() => onSelect(chat.id)}
                  tabIndex={0}
                >
                  <img
                    src={chat.avatar_url ?? '/static/img/default-avatar.svg'}
                    alt="Avatar"
                    className="avatar avatar-md"
                  />
                  <div className="chat-list-item-content">
                    <div className="chat-list-item__top">
                      <span className="chat-list-item__name">Чат с {chat.other_user_name}</span>
                      <span className="chat-list-item__time">
                        {new Date(chat.updated_at).toLocaleTimeString('ru-RU', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                    <div className="chat-list-item__preview">
                      <small className="text-muted text-truncate">
                        {chat.last_message ? chat.last_message : 'Нет сообщений'}
                      </small>
                      {chat.unread_count > 0 && (
                        <span className="badge bg-primary ms-2">{chat.unread_count}</span>
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
