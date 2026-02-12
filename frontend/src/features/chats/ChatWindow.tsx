import { useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import type { Message, WSMessageAction } from '../../shared/api/types'
import { httpPost } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'
import { MessageList } from './MessageList'
import { MessageComposer } from './MessageComposer'
import { useToast } from '../../shared/components/ToastContext'

type Props = {
  chatId?: number
  messages: Message[]
  otherUserId?: number
  otherUsername?: string
  otherAvatar?: string | null
  currentUserId?: number
  loading?: boolean
  error?: string | null
  onMessagesUpdate?: () => void
  isConnected: boolean
  isOtherUserOnline?: boolean
  send: (data: WSMessageAction) => boolean
  isMobile?: boolean
  onBackToList?: () => void
}

export function ChatWindow({
  chatId,
  messages,
  otherUserId,
  otherUsername,
  otherAvatar,
  currentUserId,
  loading,
  error,
  onMessagesUpdate,
  isConnected,
  isOtherUserOnline = false,
  send,
  isMobile,
  onBackToList,
}: Props) {
  const [messageText, setMessageText] = useState('')
  const [replyToId, setReplyToId] = useState<number | null>(null)
  const [editingMessageId, setEditingMessageId] = useState<number | null>(null)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  const pendingUploadRef = useRef<{ chatId: number; files: File[] } | null>(null)

  const { showToast } = useToast()

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    const text = messageText.trim()

    if (!chatId) {
      showToast('Выберите чат, чтобы отправлять сообщения', 'warning')
      return
    }

    if (!text && selectedFiles.length === 0) {
      showToast('Введите сообщение или прикрепите файлы', 'warning')
      return
    }

    if (!isConnected) {
      showToast('Соединение потеряно, ждём переподключения...', 'warning')
      return
    }

    if (editingMessageId) {
      const success = send({ action: 'edit', chat_id: chatId, message_id: editingMessageId, text })
      if (success) {
        setMessageText('')
        setEditingMessageId(null)
      } else {
        showToast('Не удалось отправить изменение, повторите.', 'error')
      }
      return
    }

    // Store files for upload after message is created
    if (selectedFiles.length > 0) {
      pendingUploadRef.current = { chatId, files: [...selectedFiles] }
    }

    const success = send({
      action: 'send',
      chat_id: chatId,
      text: text || ' ',
      reply_to_id: replyToId ?? undefined,
    })

    if (success) {
      setMessageText('')
      setReplyToId(null)

      // Handle file uploads - need to get message ID from the new message
      if (pendingUploadRef.current) {
        const { chatId: uploadChatId, files } = pendingUploadRef.current
        pendingUploadRef.current = null
        setSelectedFiles([])

        // Wait a bit for the message to be created, then get the last message ID
        setTimeout(async () => {
          const lastMessage = messages[messages.length - 1]
          if (lastMessage && files.length > 0) {
            setUploading(true)
            const formData = new FormData()
            files.forEach((file) => formData.append('attachments', file))

            try {
              await httpPost(endpoints.attachments.upload(uploadChatId, lastMessage.id + 1), formData)
              onMessagesUpdate?.()
            } catch (err) {
              console.error('Ошибка загрузки вложений:', err)
              showToast('Не удалось загрузить вложения', 'error')
            } finally {
              setUploading(false)
            }
          }
        }, 500)
      }
    } else {
      showToast('Не удалось отправить сообщение, повторите.', 'error')
      pendingUploadRef.current = null
    }
  }

  const handleFileSelect = (files: File[]) => {
    setSelectedFiles((prev) => [...prev, ...files])
  }

  const handleRemoveFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, idx) => idx !== index))
  }

  const handleReply = (msgId: number) => {
    setEditingMessageId(null)
    setReplyToId(msgId)
    setMessageText('')
  }

  const handleEdit = (msgId: number, text: string) => {
    setReplyToId(null)
    setEditingMessageId(msgId)
    setMessageText(text)
  }

  const handleDelete = (msgId: number) => {
    if (!confirm('Удалить сообщение?')) return
    if (!chatId) return
    send({ action: 'delete', chat_id: chatId, message_id: msgId })
  }

  const handleCancelDraft = () => {
    setReplyToId(null)
    setEditingMessageId(null)
    setMessageText('')
  }

  const emptyState = useMemo(
    () => (
      <div className="chat-window glassy empty-chat-panel">
        <div className="empty-hero">
          <div className="empty-hero__badge">Nothing</div>
          <h2>Начните новый разговор</h2>
          <p className="text-muted">Выберите контакт слева или создайте чат по username.</p>
          <div className="empty-hero__cta">
            <span className="dot online" />
            <span>Готовы к мгновенным сообщениям</span>
          </div>
        </div>
      </div>
    ),
    []
  )

  if (!chatId || !otherUsername) {
    return emptyState
  }

  return (
    <div className="chat-window glassy">
      <div className="chat-header">
        <div className="chat-header__title">
          {isMobile && (
            <button className="back-btn" onClick={onBackToList} aria-label="Назад">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M19 12H5M12 19l-7-7 7-7" />
              </svg>
            </button>
          )}
          <Link to={`/profile/${otherUserId}`} className="chat-header__link">
            <span className="avatar avatar-sm">
              <img src={otherAvatar || '/img/default-avatar.svg'} alt="avatar" />
            </span>
            <div className="chat-header__info">
              <span className="chat-peer">{otherUsername}</span>
              <div className="chat-header__meta">
                <span className={`dot ${isOtherUserOnline ? 'online' : 'offline'}`} />
                <span className="chat-subtitle">{isOtherUserOnline ? 'В сети' : 'Не в сети'}</span>
              </div>
            </div>
          </Link>
        </div>
        {!isConnected && (
          <div className="chat-header__actions">
            <span className="badge bg-warning text-dark">Reconnecting</span>
          </div>
        )}
      </div>

      <MessageList
        messages={messages}
        currentUserId={currentUserId}
        otherUsername={otherUsername}
        chatId={chatId}
        loading={loading}
        error={error}
        onReply={handleReply}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      <MessageComposer
        messages={messages}
        replyToId={replyToId}
        editingMessageId={editingMessageId}
        messageText={messageText}
        selectedFiles={selectedFiles}
        uploading={uploading}
        onMessageTextChange={setMessageText}
        onSubmit={handleSubmit}
        onFileSelect={handleFileSelect}
        onRemoveFile={handleRemoveFile}
        onCancelDraft={handleCancelDraft}
      />
    </div>
  )
}
