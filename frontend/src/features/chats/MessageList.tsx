import { useEffect, useMemo, useRef } from 'react'

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
  // Build a lookup map for group members
  const membersMap = new Map(groupMembers.map((m) => [m.user_id, m]))

  // Reverse messages for column-reverse layout:
  // CSS column-reverse flips visual order, so we reverse the array
  // to display oldest→newest (top→bottom) while scroll starts at bottom
  const reversedMessages = useMemo(() => messages.slice().reverse(), [messages])

  // Auto-scroll to newest message. Column-reverse layout means scrollTop=0 is the visual bottom.
  // Android WebView loses the column-reverse scroll anchor when the list updates, so we pin it manually:
  // - always pin when the caller sent the message (they just hit send)
  // - pin when the user is already near the bottom (live-reading)
  // - leave alone when the user scrolled up reading history (don't yank them down)
  const containerRef = useRef<HTMLDivElement>(null)
  const lastMessageId = messages.length > 0 ? messages[messages.length - 1].id : undefined
  const lastMessageOwner = messages.length > 0 ? messages[messages.length - 1].user_id : undefined
  useEffect(() => {
    const el = containerRef.current
    if (!el || lastMessageId === undefined) return
    const isOwn = lastMessageOwner === currentUserId
    // In column-reverse, scrollTop is 0 at the bottom and negative as you scroll up.
    const isNearBottom = el.scrollTop > -120
    if (isOwn || isNearBottom) {
      el.scrollTop = 0
    }
  }, [lastMessageId, lastMessageOwner, currentUserId])

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

    return reversedMessages.map((message) => {
      // Deleted messages are kept in the array so reply quotes can still resolve them,
      // but they are not rendered as standalone messages in the chat.
      if (message.is_deleted) return null

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
