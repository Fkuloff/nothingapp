import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext, useSearchParams } from 'react-router-dom'

import type { OutletContextType } from '../App'
import { useAccountKey } from '../features/auth/AccountKey'
import { useAuthContext } from '../features/auth/AuthContext'
import { useCallContext } from '../features/calls/CallContext'
import { ChatList } from '../features/chats/ChatList'
import { ChatWindow } from '../features/chats/ChatWindow'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import { clearChat, deleteChat, getChatMessages, getCurrentUserChats, getPinnedMessages, pinMessage, unpinMessage } from '../shared/api/chatsApi'
import { getGroupInfo } from '../shared/api/groupsApi'
import { getUserPresence } from '../shared/api/presenceApi'
import type { ChatItem, GroupInfoResponse, Message, PinnedMessage, WSEnvelope, WSEvent } from '../shared/api/types'
import { decryptIncomingText } from '../shared/crypto/decryptIncoming'
import { getChatKey } from '../shared/crypto/peerKeys'

// selectScheme2Payload picks (ciphertext, iv) for a scheme=2 WS broadcast. For
// 1-on-1 it's just the top-level text/iv. For group pairwise (envelopes set)
// it's the envelope addressed to the current user — if none is present the
// current user wasn't a recipient (e.g. joined the group after the message
// was sent) and we return null so the UI shows the placeholder.
function selectScheme2Payload(
  envelopes: WSEnvelope[] | undefined,
  topText: string,
  topIv: string | undefined,
  currentUserId: number | undefined,
): { text: string; iv: string | undefined } | null {
  if (!envelopes || envelopes.length === 0) {
    return { text: topText, iv: topIv }
  }
  if (!currentUserId) return null
  const own = envelopes.find((e) => e.recipient_id === currentUserId)
  if (!own) return null
  return { text: own.ciphertext, iv: own.iv }
}
import { useAndroidBack } from '../shared/hooks/useAndroidBack'
import { useGlobalWebSocket } from '../shared/hooks/useGlobalWebSocket'
import { subscribePendingChat } from '../shared/pendingChat'

const CACHED_CHATS_KEY = 'cached_chats_list'

function readCachedChats(): ChatItem[] {
  try {
    const raw = localStorage.getItem(CACHED_CHATS_KEY)
    if (!raw) return []
    const data = JSON.parse(raw) as ChatItem[]
    return Array.isArray(data) ? data : []
  } catch {
    return []
  }
}

function writeCachedChats(chats: ChatItem[]) {
  try {
    localStorage.setItem(CACHED_CHATS_KEY, JSON.stringify(chats))
  } catch {
    // ignore quota / serialization issues — caching is best-effort
  }
}

export default function ChatsPage() {
  const { setMenuOpen, onChatSelectedRef } = useOutletContext<OutletContextType>()
  const { user } = useAuthContext()
  const callContext = useCallContext()
  const accountKeyCtx = useAccountKey()
  // Snapshot the current account_key in a ref so the WS event handler (which captures
  // closures with stale `accountKeyCtx`) always sees the latest value. The handler
  // re-renders on `accountKeyCtx.state` would otherwise unsubscribe/resubscribe the
  // WS pipe each time the key changes.
  const accountKeyRef = useRef<CryptoKey | null>(null)
  useEffect(() => {
    accountKeyRef.current =
      accountKeyCtx.state.status === 'ready' ? accountKeyCtx.state.key : null
  }, [accountKeyCtx.state])
  const [searchParams, setSearchParams] = useSearchParams()
  const [chats, setChats] = useState<ChatItem[]>(() => readCachedChats())
  const [messages, setMessages] = useState<Message[]>([])
  const [activeChatId, setActiveChatId] = useState<number | null>(null)
  const [loadingChats, setLoadingChats] = useState(() => readCachedChats().length === 0)
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [chatsError, setChatsError] = useState<string | null>(null)
  const [messagesError, setMessagesError] = useState<string | null>(null)
  const [isMobile, setIsMobile] = useState(false)
  const [onlineUsers, setOnlineUsers] = useState<Set<number>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')
  const [groupInfo, setGroupInfo] = useState<GroupInfoResponse | null>(null)
  const [pinnedMessages, setPinnedMessages] = useState<PinnedMessage[]>([])
  const chatsRef = useRef<ChatItem[]>([])
  const messagesRef = useRef<Message[]>([])
  const loadChatsRef = useRef<() => void>(() => {})
  const loadMessagesRef = useRef<(chatId: number) => void>(() => {})
  const messageCacheRef = useRef<Map<number, Message[]>>(new Map())
  const handleCallEventRef = useRef(callContext.handleCallEvent)

  // Keep refs in sync for use in WS handler
  useEffect(() => {
    chatsRef.current = chats
  }, [chats])
  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

  useEffect(() => {
    handleCallEventRef.current = callContext.handleCallEvent
  }, [callContext.handleCallEvent])

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

      // Route call signaling events to CallContext
      if ('action' in event && event.action.startsWith('call_')) {
        if (event.action === 'call_offer') {
          const chat = chatsRef.current.find((c) => c.id === event.chat_id)
          handleCallEventRef.current(event, chat ? { username: chat.other_user_name || '', avatar: chat.avatar_url } : undefined)
        } else {
          handleCallEventRef.current(event)
        }
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
        messageCacheRef.current.delete(event.chat_id)
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
        // Decrypt scheme=2 payloads before they touch either the chat-list preview
        // or the message thread.
        //   - 1-on-1: chat_key = ECDH(self, peer.public_key) — both sides derive
        //     the same key by symmetry, so the same key encrypts and decrypts.
        //   - Group pairwise: the broadcast carries one envelope per recipient.
        //     Pick the envelope addressed to us, then chat_key = ECDH(self,
        //     sender.public_key). For my own outgoing message in a group this
        //     resolves to ECDH(self, self.public) via the self-envelope — same
        //     derivation rule, no special case in the receive path.
        const accountKey = accountKeyRef.current
        const chat = chatsRef.current.find((c) => c.id === chatId)
        const isGroup = chat?.is_group ?? false
        const ecdhPeerUserId = isGroup
          ? event.user_id
          : (chat?.other_user_id ?? null)

        const payload = event.scheme === 2
          ? selectScheme2Payload(event.envelopes, event.text, event.iv, user?.id)
          : { text: event.text, iv: event.iv }

        const decryptPromise = payload === null
          ? Promise.resolve({ text: '[зашифрованное сообщение]', scheme: event.scheme, iv: undefined })
          : getChatKey(accountKey, ecdhPeerUserId).then((chatKey) => decryptIncomingText(
              { text: payload.text, scheme: event.scheme, iv: payload.iv },
              chatKey,
            ))

        decryptPromise.then((decrypted) => {
          const text = decrypted.text

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
              // Preserve scheme so EditMessage knows whether to re-encrypt or not.
              scheme: event.scheme,
              reply_to_id: event.reply_to_id ?? null,
              edited_at: event.edited_at ?? null,
              is_deleted: event.is_deleted,
              created_at: event.created_at,
              attachments: event.attachments ?? [],
            }
            setMessages((prev) => {
              // Deduplicate: after a WebSocket reconnect the server replays unread messages,
              // which may already be in prev from the HTTP fetch we triggered on reconnect.
              // Blindly appending would show the same message two or three times until the
              // user re-enters the chat and the list is freshly loaded.
              if (prev.some((m) => m.id === newMessage.id)) return prev
              const next = [...prev, newMessage]
              messageCacheRef.current.set(chatId, next)
              return next
            })
          }
        }).catch((err) => console.error('decrypt new message failed:', err))
        return
      }

      if (event.action === 'attachments_added' && chatId === activeChatId) {
        setMessages((prev) => {
          const next = prev.map((msg) =>
            msg.id === event.message_id ? { ...msg, attachments: event.attachments } : msg
          )
          messageCacheRef.current.set(chatId, next)
          return next
        })
        return
      }

      if (event.action === 'edit' && chatId === activeChatId) {
        // Same envelope/sender resolution rules as the 'new' path above. For
        // group scheme=2 edits the broadcast carries a fresh envelope set;
        // for 1-on-1 it carries text+iv directly.
        const accountKey = accountKeyRef.current
        const chat = chatsRef.current.find((c) => c.id === chatId)
        const isGroup = chat?.is_group ?? false
        // Use the original sender's user_id for group; need to look it up.
        const existing = messagesRef.current.find((m) => m.id === event.id)
        const senderUserId = existing?.user_id ?? user?.id ?? 0
        const ecdhPeerUserId = isGroup
          ? senderUserId
          : (chat?.other_user_id ?? null)

        const payload = event.scheme === 2
          ? selectScheme2Payload(event.envelopes, event.text, event.iv, user?.id)
          : { text: event.text, iv: event.iv }

        const decryptPromise = payload === null
          ? Promise.resolve({ text: '[зашифрованное сообщение]', scheme: event.scheme, iv: undefined })
          : getChatKey(accountKey, ecdhPeerUserId).then((chatKey) => decryptIncomingText(
              { text: payload.text, scheme: event.scheme, iv: payload.iv },
              chatKey,
            ))

        decryptPromise.then((decrypted) => {
          setMessages((prev) => {
            const next = prev.map((msg) =>
              msg.id === event.id
                ? {
                    ...msg,
                    text: decrypted.text,
                    edited_at: event.edited_at ?? new Date().toISOString(),
                  }
                : msg,
            )
            messageCacheRef.current.set(chatId, next)
            return next
          })
        }).catch((err) => console.error('decrypt edit failed:', err))
        return
      }

      if (event.action === 'delete') {
        if (chatId === activeChatId) {
          setMessages((prev) => {
            const next = prev.map((msg) => (msg.id === event.id ? { ...msg, is_deleted: event.is_deleted } : msg))
            messageCacheRef.current.set(chatId, next)
            return next
          })
        }
        // Refresh chat list so the sidebar preview drops the deleted message's text
        // and falls back to the previous non-deleted message (or empty).
        loadChatsRef.current()
        return
      }

      if (event.action === 'chat_cleared') {
        messageCacheRef.current.delete(event.chat_id)
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
        messageCacheRef.current.delete(event.chat_id)
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

  // Register WS send function with CallContext so it can send signaling messages
  useEffect(() => {
    if (isConnected) {
      callContext.registerSend(send)
    }
    return () => callContext.registerSend(null)
  }, [isConnected, send, callContext])

  // On WebSocket reconnect (typically after the app was backgrounded and Android froze the
  // process), pull fresh state — the chat list preview + active-chat messages may be stale
  // because we missed events while offline. Skip the very first connect: that's just the
  // initial mount, and the other effects already cover it.
  const hasConnectedRef = useRef(false)
  useEffect(() => {
    if (!isConnected) return
    if (!hasConnectedRef.current) {
      hasConnectedRef.current = true
      return
    }
    loadChatsRef.current()
    if (activeChatId !== null) {
      messageCacheRef.current.delete(activeChatId)
      loadMessagesRef.current(activeChatId)
    }
  }, [isConnected, activeChatId])

  const loadChats = useCallback(async () => {
    // If we already have data (from cache or a previous load), keep showing it while
    // we refetch in the background — never flash a spinner on top of stale-but-useful data.
    const haveData = chatsRef.current.length > 0
    try {
      setChatsError(null)
      if (!haveData) setLoadingChats(true)
      const data = await getCurrentUserChats()
      // Decrypt the last-message preview for any scheme=2 chat where the server
      // attached the raw ciphertext + sender_id. 1-on-1: chat_key derived from
      // peer's public_key (sender is either us or peer, both → same key). Group:
      // chat_key derived from sender's public_key (per-user envelope already
      // resolved server-side).
      const accountKey = accountKeyRef.current
      await Promise.all(data.map(async (c) => {
        if (
          c.last_message_scheme === 2 &&
          c.last_message_ciphertext &&
          c.last_message_iv &&
          c.last_message_sender_id !== undefined
        ) {
          const ecdhPeerUserId = c.is_group
            ? c.last_message_sender_id
            : (c.other_user_id ?? null)
          const chatKey = await getChatKey(accountKey, ecdhPeerUserId)
          const decrypted = await decryptIncomingText(
            { text: c.last_message_ciphertext, scheme: 2, iv: c.last_message_iv },
            chatKey,
          )
          c.last_message = decrypted.text.trim() || c.last_message
        }
      }))
      const sortedData = data.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
      chatsRef.current = sortedData
      setChats(sortedData)
      writeCachedChats(sortedData)
    } catch (err) {
      console.error('Ошибка загрузки чатов', err)
      // Only surface the error if we have no cached data — otherwise the user sees their
      // last-known chat list, which is more useful than an error message.
      if (!haveData) {
        setChatsError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
      }
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
    // Serve from cache instantly if available
    const cached = messageCacheRef.current.get(chatId)
    if (cached) {
      setMessages(cached)
      setLoadingMessages(false)
    } else {
      setLoadingMessages(true)
    }

    // Always fetch fresh data in background
    try {
      setMessagesError(null)
      const raw = await getChatMessages(chatId)
      // For 1-on-1: derive one chat_key from the peer's public_key and decrypt
      // every scheme=2 message with it. For groups: server already resolved
      // each row's per-user envelope into text/iv, but each message may have a
      // different sender, so we derive a chat_key per unique sender_id. The
      // peerKeys cache makes the N lookups effectively free after the first.
      const chat = chatsRef.current.find((c) => c.id === chatId)
      const accountKey = accountKeyRef.current
      let data: Message[]
      if (chat?.is_group) {
        const senderKeys = new Map<number, CryptoKey | null>()
        const uniqueSenders = Array.from(new Set(raw.filter((m) => m.scheme === 2).map((m) => m.user_id)))
        await Promise.all(
          uniqueSenders.map(async (senderId) => {
            senderKeys.set(senderId, await getChatKey(accountKey, senderId))
          }),
        )
        data = await Promise.all(
          raw.map((m) => decryptIncomingText(m, m.scheme === 2 ? (senderKeys.get(m.user_id) ?? null) : null)),
        )
      } else {
        const peerUserId = chat?.other_user_id ?? null
        const chatKey = await getChatKey(accountKey, peerUserId)
        data = await Promise.all(raw.map((m) => decryptIncomingText(m, chatKey)))
      }
      messageCacheRef.current.set(chatId, data)
      setMessages(data)
    } catch (err) {
      console.error('Ошибка загрузки сообщений', err)
      if (!cached) {
        setMessages([])
        setMessagesError(err instanceof Error ? err.message : 'Не удалось загрузить сообщения')
      }
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

  // Android back: if a chat is open, deselect it (return to list); otherwise let system handle (exit app).
  useAndroidBack(() => {
    if (activeChatId !== null) {
      setActiveChatId(null)
      return true
    }
    return false
  }, true)

  // Open a chat on push notification tap. Uses a router-free pub/sub (pendingChat) because
  // HashRouter on native doesn't reliably expose search params from navigate('/?chat=X').
  // Cold-start taps are covered too — subscribePendingChat replays a value that was set
  // before ChatsPage mounted. We drop the cached messages for that chat so the subsequent
  // useEffect → loadMessages fetches fresh from the server (otherwise we'd flash stale
  // cached messages and the message that triggered the push wouldn't appear until the
  // background fetch completed).
  useEffect(() => subscribePendingChat((id) => {
    messageCacheRef.current.delete(id)
    setActiveChatId(id)
    loadChatsRef.current()
  }), [])

  // Web browser path: ?chat= in the actual URL (BrowserRouter)
  useEffect(() => {
    const chatId = searchParams.get('chat')
    if (chatId) {
      setActiveChatId(Number(chatId))
      searchParams.delete('chat')
      setSearchParams(searchParams, { replace: true })
    }
  }, [searchParams, setSearchParams])

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
      messageCacheRef.current.delete(chatId)
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
      messageCacheRef.current.delete(chatId)
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
      messageCacheRef.current.delete(activeChatId)
      setChats((prev) => prev.filter((c) => c.id !== activeChatId))
      setActiveChatId(null)
      setMessages([])
      setGroupInfo(null)
      setPinnedMessages([])
    }
  }, [activeChatId])

  const handleGroupLeft = useCallback(() => {
    if (activeChatId) {
      messageCacheRef.current.delete(activeChatId)
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
