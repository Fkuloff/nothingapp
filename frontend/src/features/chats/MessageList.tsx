import { useEffect, useRef } from 'react'

import type { GroupMember, Message } from '../../shared/api/types'
import { MessageItem } from './MessageItem'
import { SystemMessage } from './SystemMessage'

type Props = {
  messages: Message[]
  currentUserId?: number
  otherUsername: string
  isGroup?: boolean
  groupMembers?: GroupMember[]
  loading?: boolean
  error?: string | null
  pinnedMessageIds?: Set<number>
  canPin?: boolean
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
  onPin?: (msgId: number) => void
  onUnpin?: (msgId: number) => void
}

// Deterministic color palette for group sender names
const SENDER_COLORS = [
  '#e57373', '#f06292', '#ba68c8', '#9575cd',
  '#7986cb', '#64b5f6', '#4fc3f7', '#4dd0e1',
  '#4db6ac', '#81c784', '#aed581', '#ff8a65',
]

function getSenderColor(userId: number): string {
  return SENDER_COLORS[userId % SENDER_COLORS.length]
}

function getSenderName(
  message: Message,
  currentUserId: number | undefined,
  otherUsername: string,
  isGroup: boolean,
  membersMap: Map<number, GroupMember>,
): string {
  if (message.user_id === currentUserId) return 'Вы'
  if (isGroup) {
    const member = membersMap.get(message.user_id)
    return member?.name || member?.username || `User #${message.user_id}`
  }
  return otherUsername
}

export function MessageList({
  messages,
  currentUserId,
  otherUsername,
  isGroup = false,
  groupMembers = [],
  loading,
  error,
  pinnedMessageIds,
  canPin,
  onReply,
  onEdit,
  onDelete,
  onPin,
  onUnpin,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const prevChatIdRef = useRef<number | undefined>(undefined)
  const prevMessagesLengthRef = useRef<number>(0)
  const hasScrolledRef = useRef(false)

  // Build a lookup map for group members
  const membersMap = new Map(groupMembers.map((m) => [m.user_id, m]))

  // Derive chatId from first message (for scroll tracking)
  const chatId = messages.length > 0 ? messages[0].chat_id : undefined

  // Reset scroll flag when chat changes
  useEffect(() => {
    if (prevChatIdRef.current !== chatId) {
      hasScrolledRef.current = false
    }
  }, [chatId])

  // Scroll to bottom after messages load
  useEffect(() => {
    if (!containerRef.current || loading || messages.length === 0) return

    const isNewChat = prevChatIdRef.current !== chatId
    const hasNewMessages = messages.length > prevMessagesLengthRef.current

    const scrollToBottom = () => {
      if (containerRef.current) {
        containerRef.current.scrollTop = containerRef.current.scrollHeight
      }
    }

    if (isNewChat && !hasScrolledRef.current) {
      setTimeout(() => {
        scrollToBottom()
        hasScrolledRef.current = true
      }, 50)
    } else if (hasNewMessages && hasScrolledRef.current) {
      containerRef.current.scrollTo({
        top: containerRef.current.scrollHeight,
        behavior: 'smooth'
      })
    }

    prevChatIdRef.current = chatId
    prevMessagesLengthRef.current = messages.length
  }, [chatId, messages, loading])

  const renderContent = () => {
    if (error) {
      return <div className="alert alert-danger m-3">{error}</div>
    }

    if (loading) {
      return <div className="text-center p-4 text-muted">Загружаем сообщения...</div>
    }

    if (messages.length === 0) {
      return <div className="text-center text-muted p-4">В этом чате пока нет сообщений.</div>
    }

    return messages.map((message) => {
      if (message.type === 'system') {
        return <SystemMessage key={message.id} text={message.text} />
      }

      const isOwn = message.user_id === currentUserId
      const senderName = getSenderName(message, currentUserId, otherUsername, isGroup, membersMap)
      const replyToMessage = message.reply_to_id
        ? messages.find((m) => m.id === message.reply_to_id)
        : null

      return (
        <MessageItem
          key={message.id}
          message={message}
          isOwn={isOwn}
          senderName={senderName}
          senderColor={isGroup && !isOwn ? getSenderColor(message.user_id) : undefined}
          replyToMessage={replyToMessage}
          isPinned={pinnedMessageIds?.has(message.id)}
          canPin={canPin}
          onReply={onReply}
          onEdit={onEdit}
          onDelete={onDelete}
          onPin={onPin}
          onUnpin={onUnpin}
        />
      )
    })
  }

  return (
    <div id="messages" className="chat-body fancy-scroll" ref={containerRef}>
      {renderContent()}
    </div>
  )
}
