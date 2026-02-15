import { useMemo } from 'react'

import type { ChatItem } from '../../shared/api/types'

type Props = {
  chats: ChatItem[]
  activeChatId?: number
  onSelect: (chatId: number) => void
  loading?: boolean
  error?: string | null
}

export function ChatList({ chats, activeChatId, onSelect, loading, error }: Props) {
  const sortedChats = useMemo(
    () => [...chats].sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
    [chats]
  )

  return (
    <div className="telegram-chat-list">
      {error && <div className="alert alert-danger alert-sm m-2">{error}</div>}

      {/* Chat list items */}
      <div className="telegram-chat-list__items fancy-scroll">
        {loading ? (
          <p className="text-muted small p-3">Загружаем диалоги...</p>
        ) : sortedChats.length === 0 ? (
          <div className="telegram-empty-list">
            <p>Пока пусто</p>
            <p className="text-muted small">Добавьте контакт и начните общение</p>
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
