import { useEffect, useRef } from 'react'
import type { Message } from '../../shared/api/types'
import { MessageItem } from './MessageItem'

type Props = {
  messages: Message[]
  currentUserId?: number
  otherUserId?: number
  otherUsername: string
  chatId?: number
  loading?: boolean
  error?: string | null
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
}

export function MessageList({
  messages,
  currentUserId,
  otherUserId,
  otherUsername,
  chatId,
  loading,
  error,
  onReply,
  onEdit,
  onDelete,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const prevChatIdRef = useRef<number | undefined>(undefined)
  const prevMessagesLengthRef = useRef<number>(0)
  const hasScrolledRef = useRef(false)

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
      // For new chat: use setTimeout to ensure DOM is fully rendered
      setTimeout(() => {
        scrollToBottom()
        hasScrolledRef.current = true
      }, 50)
    } else if (hasNewMessages && hasScrolledRef.current) {
      // For new messages in current chat: smooth scroll
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
      const isOwn = message.user_id === currentUserId
      const replyToMessage = message.reply_to_id
        ? messages.find((m) => m.id === message.reply_to_id)
        : null

      return (
        <MessageItem
          key={message.id}
          message={message}
          isOwn={isOwn}
          senderName={isOwn ? 'Вы' : otherUsername}
          replyToMessage={replyToMessage}
          onReply={onReply}
          onEdit={onEdit}
          onDelete={onDelete}
          chatId={chatId}
          otherUserId={otherUserId}
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
