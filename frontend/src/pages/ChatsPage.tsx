import { useCallback, useEffect, useMemo, useState } from 'react'
import { ChatList } from '../features/chats/ChatList'
import { ChatWindow } from '../features/chats/ChatWindow'
import type { ChatItem, Message } from '../shared/api/types'
import { useAuthContext } from '../features/auth/AuthContext'
import { getCurrentUserChats, getChatMessages } from '../shared/api/chatsApi'

export default function ChatsPage() {
  const { user } = useAuthContext()
  const [chats, setChats] = useState<ChatItem[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [activeChatId, setActiveChatId] = useState<number | null>(null)
  const [loadingChats, setLoadingChats] = useState(true)
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [chatsError, setChatsError] = useState<string | null>(null)
  const [messagesError, setMessagesError] = useState<string | null>(null)
  const [isMobile, setIsMobile] = useState(false)

  useEffect(() => {
    const computeIsMobile = () => {
      const byWidth = window.matchMedia('(max-width: 900px)').matches
      const byUA = /Mobi|Android|iPhone|iPad/i.test(window.navigator.userAgent)
      setIsMobile(byWidth || byUA)
    }

    computeIsMobile()
    window.addEventListener('resize', computeIsMobile)
    return () => window.removeEventListener('resize', computeIsMobile)
  }, [])

  const loadChats = useCallback(async () => {
    try {
      setChatsError(null)
      setLoadingChats(true)
      const data = await getCurrentUserChats()
      setChats(data)

      // Если текущий активный чат исчез, сбрасываем выбор
      if (activeChatId && !data.find((c) => c.id === activeChatId)) {
        setActiveChatId(null)
        setMessages([])
      }
    } catch (err) {
      console.error('Ошибка загрузки чатов', err)
      setChatsError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
    } finally {
      setLoadingChats(false)
    }
  }, [activeChatId])

  const loadMessages = useCallback(async (chatId: number) => {
    try {
      setMessagesError(null)
      setLoadingMessages(true)
      const data = await getChatMessages(chatId)
      setMessages(data)
    } catch (err) {
      console.error('Ошибка загрузки сообщений', err)
      setMessages([])
      setMessagesError(err instanceof Error ? err.message : 'Не удалось загрузить сообщения')
    } finally {
      setLoadingMessages(false)
    }
  }, [])

  useEffect(() => {
    loadChats()
  }, [loadChats])

  useEffect(() => {
    if (activeChatId) {
      loadMessages(activeChatId)
    }
  }, [activeChatId, loadMessages])

  const handleChatCreated = () => {
    loadChats()
  }

  const handleMessagesUpdate = () => {
    if (activeChatId) {
      loadMessages(activeChatId)
    }
  }

  const activeChat = useMemo(
    () => chats.find((chat) => chat.id === activeChatId),
    [activeChatId, chats]
  )

  return (
    <div className={`workspace${isMobile ? ' mobile' : ''}${isMobile && activeChatId ? ' chat-active' : ''}`}>
      <div className="workspace__panel">
        <ChatList
          chats={chats}
          activeChatId={activeChatId ?? undefined}
          onSelect={(id) => setActiveChatId(id)}
          onChatCreated={handleChatCreated}
          loading={loadingChats}
          error={chatsError}
        />
      </div>

      <div className="workspace__content">
        <ChatWindow
          chatId={activeChatId ?? undefined}
          messages={messages}
          otherUsername={activeChat?.other_user_name}
          otherAvatar={activeChat?.avatar_url}
          currentUserId={user?.id}
          loading={loadingMessages}
          error={messagesError}
          onMessagesUpdate={handleMessagesUpdate}
          onRefreshChats={loadChats}
          isMobile={isMobile}
          onBackToList={() => setActiveChatId(null)}
        />
      </div>
    </div>
  )
}
