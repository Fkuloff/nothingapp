import { useMemo } from 'react'

import { resolveApiUrl } from '../../shared/api/httpClient'
import type { ChatItem } from '../../shared/api/types'
import { BookmarkIcon, GroupIcon } from '../../shared/components/Icons'

type Props = {
  chats: ChatItem[]
  activeChatId?: number
  onSelect: (chatId: number) => void
  loading?: boolean
  error?: string | null
}

function getChatDisplayName(chat: ChatItem): string {
  if (chat.is_favorites) return 'Избранное'
  if (chat.is_group) return chat.group_name || 'Группа'
  return chat.other_user_name || 'Чат'
}

export function ChatList({ chats, activeChatId, onSelect, loading, error }: Props) {
  // Favorites is always pinned to the top regardless of last activity; remaining
  // chats stay in updated_at-desc order (matches the server's bucketing).
  const sortedChats = useMemo(
    () =>
      [...chats].sort((a, b) => {
        if (a.is_favorites && !b.is_favorites) return -1
        if (!a.is_favorites && b.is_favorites) return 1
        return b.updated_at.localeCompare(a.updated_at)
      }),
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
              const displayName = getChatDisplayName(chat)

              return (
                <li
                  key={chat.id}
                  className={`chat-list-item${isActive ? ' active' : ''}${hasUnread ? ' has-unread' : ''}`}
                  role="button"
                  onClick={() => onSelect(chat.id)}
                  tabIndex={0}
                >
                  <span className="avatar avatar-md">
                    {chat.is_favorites ? (
                      <span
                        className="avatar avatar-md d-flex align-items-center justify-content-center"
                        style={{ background: 'var(--bs-primary, #2481cc)', color: '#fff', width: '100%', height: '100%' }}
                      >
                        <BookmarkIcon size={22} />
                      </span>
                    ) : (
                      <img src={resolveApiUrl(chat.avatar_url) || '/img/default-avatar.svg'} alt="Avatar" loading="lazy" />
                    )}
                  </span>
                  <div className="chat-list-item-content">
                    <div className="chat-list-item__top">
                      <span className="chat-list-item__name">
                        {chat.is_group && <GroupIcon className="chat-list-item__group-icon" size={14} />}
                        {displayName}
                      </span>
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
