import { useEffect, useRef } from 'react'
import type { Message } from '../../shared/api/types'
import { MessageItem } from './MessageItem'

type Props = {
  messages: Message[]
  currentUserId?: number
  otherUsername: string
  loading?: boolean
  error?: string | null
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
}

export function MessageList({
  messages,
  currentUserId,
  otherUsername,
  loading,
  error,
  onReply,
  onEdit,
  onDelete,
}: Props) {
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  if (error) {
    return (
      <div id="messages" className="chat-body fancy-scroll">
        <div className="alert alert-danger m-3">{error}</div>
      </div>
    )
  }

  if (loading) {
    return (
      <div id="messages" className="chat-body fancy-scroll">
        <div className="text-center p-4 text-muted">Загружаем сообщения...</div>
      </div>
    )
  }

  if (messages.length === 0) {
    return (
      <div id="messages" className="chat-body fancy-scroll">
        <div className="text-center text-muted p-4">В этом чате пока нет сообщений.</div>
      </div>
    )
  }

  return (
    <div id="messages" className="chat-body fancy-scroll">
      {messages.map((message) => {
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
          />
        )
      })}
      <div ref={messagesEndRef} />
    </div>
  )
}
