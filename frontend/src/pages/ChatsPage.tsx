import { useCallback, useEffect, useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { ChatList } from '../features/chats/ChatList'
import { ChatWindow } from '../features/chats/ChatWindow'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import type { ChatItem, Message, WSEvent } from '../shared/api/types'
import { useAuthContext } from '../features/auth/AuthContext'
import { getCurrentUserChats, getChatMessages } from '../shared/api/chatsApi'
import { getUserPresence } from '../shared/api/presenceApi'
import { useGlobalWebSocket } from '../shared/hooks/useGlobalWebSocket'
import type { OutletContextType } from '../App'

export default function ChatsPage() {
  const { setMenuOpen } = useOutletContext<OutletContextType>()
  const { user } = useAuthContext()
  const [chats, setChats] = useState<ChatItem[]>([])
  const [messages, setMessages] = useState<Message[]>([])
  const [activeChatId, setActiveChatId] = useState<number | null>(null)
  const [loadingChats, setLoadingChats] = useState(true)
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [chatsError, setChatsError] = useState<string | null>(null)
  const [messagesError, setMessagesError] = useState<string | null>(null)
  const [isMobile, setIsMobile] = useState(false)
  const [onlineUsers, setOnlineUsers] = useState<Set<number>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    const computeIsMobile = () => {
      const byWidth = window.matchMedia('(max-width: 768px)').matches
      const byUA = /Mobi|Android|iPhone|iPad/i.test(window.navigator.userAgent)
      setIsMobile(byWidth || byUA)
    }

    computeIsMobile()
    window.addEventListener('resize', computeIsMobile)
    return () => window.removeEventListener('resize', computeIsMobile)
  }, [])

  // Handle WebSocket messages globally
  const handleWebSocketMessage = useCallback(
    (event: WSEvent) => {
      if ('error' in event) {
        console.error('WebSocket error:', event.error)
        return
      }

      // Handle presence changes
      if (event.action === 'presence_changed') {
        setOnlineUsers((prev) => {
          const next = new Set(prev)
          if (event.is_online) {
            next.add(event.user_id)
          } else {
            next.delete(event.user_id)
          }
          return next
        })
        return
      }

      const chatId = event.chat_id

      if (event.action === 'new') {
        // Update chat list reactively
        setChats((prevChats) => {
          const chatIndex = prevChats.findIndex((c) => c.id === chatId)
          if (chatIndex === -1) {
            // New chat - reload full list
            getCurrentUserChats().then(setChats).catch(console.error)
            return prevChats
          }

          const updatedChats = [...prevChats]
          const chat = { ...updatedChats[chatIndex] }

          // Update last message preview
          chat.last_message = event.text || '[Вложение]'
          chat.updated_at = event.created_at

          // Increment unread count if not active chat and not own message
          if (chatId !== activeChatId && event.user_id !== user?.id) {
            chat.unread_count = (chat.unread_count || 0) + 1
          }

          updatedChats[chatIndex] = chat

          // Sort by updated_at (most recent first)
          return updatedChats.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
        })

        // Add message to active chat
        if (chatId === activeChatId) {
          const newMessage: Message = {
            id: event.id,
            chat_id: event.chat_id,
            user_id: event.user_id,
            text: event.text,
            reply_to_id: event.reply_to_id ?? null,
            edited_at: event.edited_at ?? null,
            is_deleted: event.is_deleted,
            created_at: event.created_at,
            attachments: [],
          }
          setMessages((prev) => [...prev, newMessage])
        }
        return
      }

      if (event.action === 'edit' && chatId === activeChatId) {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === event.id
              ? { ...msg, text: event.text, edited_at: event.edited_at ?? new Date().toISOString() }
              : msg
          )
        )
        return
      }

      if (event.action === 'delete' && chatId === activeChatId) {
        setMessages((prev) =>
          prev.map((msg) => (msg.id === event.id ? { ...msg, is_deleted: event.is_deleted } : msg))
        )
      }
    },
    [activeChatId, user?.id]
  )

  const { isConnected, send } = useGlobalWebSocket({
    onMessage: handleWebSocketMessage,
    enabled: Boolean(user),
  })

  const loadChats = useCallback(async () => {
    try {
      setChatsError(null)
      setLoadingChats(true)
      const data = await getCurrentUserChats()
      setChats(data)

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
      // Clear unread count locally and notify server
      setChats((prevChats) =>
        prevChats.map((chat) =>
          chat.id === activeChatId ? { ...chat, unread_count: 0 } : chat
        )
      )
      // Mark messages as read on server
      if (isConnected) {
        send({ action: 'mark_read', chat_id: activeChatId })
      }
    }
  }, [activeChatId, loadMessages, isConnected, send])

  const handleChatCreated = () => {
    loadChats()
  }

  const handleMessagesUpdate = useCallback(() => {
    if (activeChatId) {
      loadMessages(activeChatId)
    }
  }, [activeChatId, loadMessages])

  const activeChat = useMemo(
    () => chats.find((chat) => chat.id === activeChatId),
    [activeChatId, chats]
  )

  // Load presence status when active chat changes
  useEffect(() => {
    if (activeChat?.other_user_id) {
      getUserPresence(activeChat.other_user_id)
        .then((presence) => {
          setOnlineUsers((prev) => {
            const next = new Set(prev)
            if (presence.is_online) {
              next.add(presence.user_id)
            } else {
              next.delete(presence.user_id)
            }
            return next
          })
        })
        .catch((err) => {
          console.error('Failed to load user presence:', err)
        })
    }
  }, [activeChat?.other_user_id])

  const isOtherUserOnline = useMemo(
    () => (activeChat ? onlineUsers.has(activeChat.other_user_id) : false),
    [activeChat, onlineUsers]
  )

  // Calculate total unread count
  const totalUnread = useMemo(
    () => chats.reduce((sum, chat) => sum + (chat.unread_count || 0), 0),
    [chats]
  )

  // Filter chats by search query
  const filteredChats = useMemo(() => {
    if (!searchQuery.trim()) return chats
    const query = searchQuery.toLowerCase()
    return chats.filter((chat) =>
      chat.other_user_name.toLowerCase().includes(query)
    )
  }, [chats, searchQuery])

  return (
    <div className={`telegram-layout${isMobile && activeChatId ? ' chat-active' : ''}`}>
      {/* Sidebar with chat list */}
      <div className="telegram-sidebar">
        <div className="telegram-sidebar__header">
          <HamburgerButton onClick={() => setMenuOpen(true)} />
          <div className="telegram-sidebar__search">
            <input
              type="search"
              className="form-control"
              placeholder="Поиск..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>
        <div className="telegram-sidebar__content">
          <ChatList
            chats={filteredChats}
            activeChatId={activeChatId ?? undefined}
            onSelect={(id) => setActiveChatId(id)}
            onChatCreated={handleChatCreated}
            loading={loadingChats}
            error={chatsError}
            totalUnread={totalUnread}
          />
        </div>
      </div>

      {/* Main chat area */}
      <div className="telegram-chat-area">
        <ChatWindow
          chatId={activeChatId ?? undefined}
          messages={messages}
          otherUserId={activeChat?.other_user_id}
          otherUsername={activeChat?.other_user_name}
          otherAvatar={activeChat?.avatar_url}
          currentUserId={user?.id}
          loading={loadingMessages}
          error={messagesError}
          onMessagesUpdate={handleMessagesUpdate}
          isConnected={isConnected}
          isOtherUserOnline={isOtherUserOnline}
          send={send}
          isMobile={isMobile}
          onBackToList={() => setActiveChatId(null)}
        />
      </div>
    </div>
  )
}
