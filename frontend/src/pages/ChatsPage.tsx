import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router-dom'

import type { OutletContextType } from '../App'
import { useAuthContext } from '../features/auth/AuthContext'
import { ChatList } from '../features/chats/ChatList'
import { ChatWindow } from '../features/chats/ChatWindow'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import { clearChat, deleteChat, getChatMessages, getCurrentUserChats, getPinnedMessages, pinMessage, unpinMessage } from '../shared/api/chatsApi'
import { getGroupInfo } from '../shared/api/groupsApi'
import { getUserPresence } from '../shared/api/presenceApi'
import type { ChatItem, GroupInfoResponse, Message, PinnedMessage, WSEvent } from '../shared/api/types'
import { useGlobalWebSocket } from '../shared/hooks/useGlobalWebSocket'

export default function ChatsPage() {
  const { setMenuOpen, onChatSelectedRef } = useOutletContext<OutletContextType>()
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
  const [groupInfo, setGroupInfo] = useState<GroupInfoResponse | null>(null)
  const [pinnedMessages, setPinnedMessages] = useState<PinnedMessage[]>([])
  const chatsRef = useRef<ChatItem[]>([])
  const loadChatsRef = useRef<() => void>(() => {})
  const loadMessagesRef = useRef<(chatId: number) => void>(() => {})

  // Keep chatsRef in sync for use in WS handler
  useEffect(() => {
    chatsRef.current = chats
  }, [chats])

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

      // Handle group events
      if (
        event.action === 'member_added' ||
        event.action === 'member_removed' ||
        event.action === 'member_left' ||
        event.action === 'group_updated' ||
        event.action === 'role_changed'
      ) {
        // Refresh group info and messages if viewing this group
        if (event.chat_id === activeChatId) {
          getGroupInfo(event.chat_id).then(setGroupInfo).catch(console.error)
          loadMessagesRef.current(event.chat_id)
        }
        // Reload chat list to reflect name/member changes
        loadChatsRef.current()
        return
      }

      if (event.action === 'group_deleted') {
        setChats((prev) => prev.filter((c) => c.id !== event.chat_id))
        if (event.chat_id === activeChatId) {
          setActiveChatId(null)
          setMessages([])
          setGroupInfo(null)
          setPinnedMessages([])
        }
        return
      }

      // Handle pin/unpin events
      if (event.action === 'message_pinned' || event.action === 'message_unpinned') {
        if (event.chat_id === activeChatId) {
          getPinnedMessages(event.chat_id).then(setPinnedMessages).catch(console.error)
          loadMessagesRef.current(event.chat_id)
        }
        return
      }

      const chatId = 'chat_id' in event ? event.chat_id : undefined
      if (!chatId) return

      if (event.action === 'new') {
        const text = event.text

        // Update chat list reactively
        setChats((prevChats) => {
          const chatIndex = prevChats.findIndex((c) => c.id === chatId)
          if (chatIndex === -1) {
            // New chat — reload
            loadChatsRef.current()
            return prevChats
          }

          const updatedChats = [...prevChats]
          const chat = { ...updatedChats[chatIndex] }
          chat.last_message = text.trim() || '[Вложение]'
          chat.updated_at = event.created_at

          if (chatId !== activeChatId && event.user_id !== user?.id) {
            chat.unread_count = (chat.unread_count || 0) + 1
          }

          updatedChats[chatIndex] = chat
          return updatedChats.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
        })

        // Add message to active chat
        if (chatId === activeChatId) {
          const newMessage: Message = {
            id: event.id,
            chat_id: event.chat_id,
            user_id: event.user_id,
            text,
            reply_to_id: event.reply_to_id ?? null,
            edited_at: event.edited_at ?? null,
            is_deleted: event.is_deleted,
            created_at: event.created_at,
            attachments: event.attachments ?? [],
          }
          setMessages((prev) => [...prev, newMessage])
        }
        return
      }

      if (event.action === 'attachments_added' && chatId === activeChatId) {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === event.message_id ? { ...msg, attachments: event.attachments } : msg
          )
        )
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
        return
      }

      if (event.action === 'chat_cleared') {
        if (event.chat_id === activeChatId) {
          setMessages([])
          setPinnedMessages([])
        }
        setChats((prev) =>
          prev.map((c) =>
            c.id === event.chat_id ? { ...c, last_message: '', unread_count: 0 } : c
          )
        )
        return
      }

      if (event.action === 'chat_deleted') {
        setChats((prev) => prev.filter((c) => c.id !== event.chat_id))
        if (event.chat_id === activeChatId) {
          setActiveChatId(null)
          setMessages([])
          setPinnedMessages([])
        }
        return
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
      const sortedData = data.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
      chatsRef.current = sortedData
      setChats(sortedData)
    } catch (err) {
      console.error('Ошибка загрузки чатов', err)
      setChatsError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
    } finally {
      setLoadingChats(false)
    }
  }, [])
  loadChatsRef.current = loadChats

  // Register chat-selected callback for SlideMenu (contacts + group creation)
  useEffect(() => {
    if (!onChatSelectedRef) return
    onChatSelectedRef.current = async (chatId) => {
      await loadChats()
      setActiveChatId(chatId)
    }
    return () => { onChatSelectedRef.current = null }
  }, [onChatSelectedRef, loadChats])

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
  loadMessagesRef.current = loadMessages

  // Handle notification click (from service worker)
  useEffect(() => {
    const handleSWMessage = (event: MessageEvent) => {
      if (event.data?.type === 'NOTIFICATION_CLICK' && event.data.chat_id) {
        setActiveChatId(event.data.chat_id)
      }
    }

    navigator.serviceWorker?.addEventListener('message', handleSWMessage)
    return () => {
      navigator.serviceWorker?.removeEventListener('message', handleSWMessage)
    }
  }, [])

  // Handle ?chat= URL parameter (opened from notification in new window)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const chatId = params.get('chat')
    if (chatId) {
      setActiveChatId(Number(chatId))
      window.history.replaceState({}, '', '/')
    }
  }, [])

  useEffect(() => {
    loadChats()
  }, [loadChats])

  // Load messages, pins, and group info when active chat changes
  useEffect(() => {
    if (activeChatId) {
      loadMessages(activeChatId)
      getPinnedMessages(activeChatId).then(setPinnedMessages).catch(console.error)

      // Load group info if it's a group chat
      const chat = chatsRef.current.find((c) => c.id === activeChatId)
      if (chat?.is_group) {
        getGroupInfo(activeChatId).then(setGroupInfo).catch(console.error)
      } else {
        setGroupInfo(null)
      }

      // Clear unread count locally
      setChats((prevChats) =>
        prevChats.map((c) =>
          c.id === activeChatId ? { ...c, unread_count: 0 } : c
        )
      )
    } else {
      setGroupInfo(null)
      setPinnedMessages([])
    }
  }, [activeChatId, loadMessages])

  // Notify server about read status (separate effect to avoid reloading on WS reconnect)
  useEffect(() => {
    if (activeChatId && isConnected) {
      send({ action: 'mark_read', chat_id: activeChatId })
    }
  }, [activeChatId, isConnected, send])

  const handleClearChat = useCallback(async (chatId: number) => {
    try {
      await clearChat(chatId)
      if (activeChatId === chatId) {
        setMessages([])
      }
      setChats((prev) =>
        prev.map((c) => c.id === chatId ? { ...c, last_message: '' } : c)
      )
    } catch (err) {
      console.error('Failed to clear chat:', err)
    }
  }, [activeChatId])

  const handleDeleteChat = useCallback(async (chatId: number) => {
    try {
      await deleteChat(chatId)
      setChats((prev) => prev.filter((c) => c.id !== chatId))
      if (activeChatId === chatId) {
        setActiveChatId(null)
        setMessages([])
      }
    } catch (err) {
      console.error('Failed to delete chat:', err)
    }
  }, [activeChatId])

  const handleGroupUpdated = useCallback(() => {
    if (activeChatId) {
      getGroupInfo(activeChatId).then(setGroupInfo).catch(console.error)
      loadMessages(activeChatId)
    }
    loadChats()
  }, [activeChatId, loadChats, loadMessages])

  const handleGroupDeleted = useCallback(() => {
    if (activeChatId) {
      setChats((prev) => prev.filter((c) => c.id !== activeChatId))
      setActiveChatId(null)
      setMessages([])
      setGroupInfo(null)
      setPinnedMessages([])
    }
  }, [activeChatId])

  const handleGroupLeft = useCallback(() => {
    if (activeChatId) {
      setChats((prev) => prev.filter((c) => c.id !== activeChatId))
      setActiveChatId(null)
      setMessages([])
      setGroupInfo(null)
      setPinnedMessages([])
    }
  }, [activeChatId])

  const handlePinMessage = useCallback(async (chatId: number, messageId: number) => {
    try {
      await pinMessage(chatId, messageId)
    } catch (err) {
      console.error('Failed to pin message:', err)
    }
  }, [])

  const handleUnpinMessage = useCallback(async (chatId: number, messageId: number) => {
    try {
      await unpinMessage(chatId, messageId)
    } catch (err) {
      console.error('Failed to unpin message:', err)
    }
  }, [])

  const activeChat = useMemo(
    () => chats.find((chat) => chat.id === activeChatId),
    [activeChatId, chats]
  )

  // Load presence status when active chat changes (1-on-1 only)
  const activeChatOtherUserId = activeChat?.other_user_id
  const activeChatIsGroup = activeChat?.is_group
  useEffect(() => {
    if (activeChatOtherUserId && !activeChatIsGroup) {
      getUserPresence(activeChatOtherUserId)
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
  }, [activeChatOtherUserId, activeChatIsGroup])

  const isOtherUserOnline = useMemo(
    () => (activeChat && !activeChat.is_group && activeChat.other_user_id ? onlineUsers.has(activeChat.other_user_id) : false),
    [activeChat, onlineUsers]
  )

  const canPin = useMemo(() => {
    if (!activeChat || !user) return false
    if (!activeChat.is_group) return true
    const member = groupInfo?.members.find((m) => m.user_id === user.id)
    return member?.role === 'admin' || member?.role === 'creator'
  }, [activeChat, user, groupInfo])

  // Filter chats by search query
  const filteredChats = useMemo(() => {
    if (!searchQuery.trim()) return chats
    const query = searchQuery.toLowerCase()
    return chats.filter((chat) => {
      if (chat.is_group) {
        return (chat.group_name || '').toLowerCase().includes(query)
      }
      return (chat.other_user_name || '').toLowerCase().includes(query)
    })
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
            loading={loadingChats}
            error={chatsError}
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
          isConnected={isConnected}
          isOtherUserOnline={isOtherUserOnline}
          send={send}
          isMobile={isMobile}
          onBackToList={() => setActiveChatId(null)}
          onClearChat={handleClearChat}
          onDeleteChat={handleDeleteChat}
          isGroup={activeChat?.is_group}
          groupName={activeChat?.is_group ? (groupInfo?.name || activeChat?.group_name) : undefined}
          groupMembers={groupInfo?.members}
          onGroupUpdated={handleGroupUpdated}
          onGroupDeleted={handleGroupDeleted}
          onGroupLeft={handleGroupLeft}
          pinnedMessages={pinnedMessages}
          canPin={canPin}
          onPinMessage={handlePinMessage}
          onUnpinMessage={handleUnpinMessage}
        />
      </div>
    </div>
  )
}
