import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Message, WSEvent } from '../../shared/api/types'
import { useWebSocket } from '../../shared/hooks/useWebSocket'
import { httpPost } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'
import { MessageList } from './MessageList'
import { MessageComposer } from './MessageComposer'
import { useToast } from '../../shared/components/ToastContext'

type Props = {
  chatId?: number
  messages: Message[]
  otherUsername?: string
  otherAvatar?: string | null
  currentUserId?: number
  loading?: boolean
  error?: string | null
  onMessagesUpdate?: () => void
  onRefreshChats?: () => void
  isMobile?: boolean
  onBackToList?: () => void
}

export function ChatWindow({
  chatId,
  messages: initialMessages,
  otherUsername,
  otherAvatar,
  currentUserId,
  loading,
  error,
  onMessagesUpdate,
  onRefreshChats,
  isMobile,
  onBackToList,
}: Props) {
  const [messages, setMessages] = useState<Message[]>(initialMessages)
  const [messageText, setMessageText] = useState('')
  const [replyToId, setReplyToId] = useState<number | null>(null)
  const [editingMessageId, setEditingMessageId] = useState<number | null>(null)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)

  const { showToast } = useToast()
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setMessages(initialMessages)
  }, [initialMessages])

  const handleWebSocketMessage = useCallback(
    (event: WSEvent) => {
      if ('error' in event) {
        console.error('WebSocket error:', event.error)
        return
      }

      if (event.action === 'new') {
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

        if (event.user_id === currentUserId) {
          setMessageText('')
          setReplyToId(null)
          setEditingMessageId(null)

          const filesToUpload = selectedFiles
          if (filesToUpload.length > 0 && chatId) {
            setUploading(true)
            const formData = new FormData()
            filesToUpload.forEach((file) => formData.append('attachments', file))

            httpPost(endpoints.attachments.upload(chatId, event.id), formData)
              .then(() => {
                setSelectedFiles([])
                if (fileInputRef.current) {
                  fileInputRef.current.value = ''
                }
                onMessagesUpdate?.()
              })
              .catch((uploadError) => {
                console.error('Ошибка загрузки вложений:', uploadError)
                showToast('Не удалось загрузить вложения', 'error')
              })
              .finally(() => {
                setUploading(false)
              })
          }
        }

        onMessagesUpdate?.()
        onRefreshChats?.()
        return
      }

      if (event.action === 'edit') {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === event.id ? { ...msg, text: event.text, edited_at: event.edited_at ?? new Date().toISOString() } : msg
          )
        )
        if (editingMessageId === event.id) {
          setMessageText('')
          setEditingMessageId(null)
        }
        onMessagesUpdate?.()
        return
      }

      if (event.action === 'delete') {
        setMessages((prev) =>
          prev.map((msg) => (msg.id === event.id ? { ...msg, is_deleted: event.is_deleted } : msg))
        )
        onMessagesUpdate?.()
      }
    },
    [chatId, currentUserId, editingMessageId, onMessagesUpdate, onRefreshChats, selectedFiles, showToast]
  )

  const { isConnected, send } = useWebSocket({
    chatId: chatId ?? 0,
    onMessage: handleWebSocketMessage,
    enabled: Boolean(chatId),
  })

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
      if (!success) {
        showToast('Не удалось отправить изменение, повторите.', 'error')
      }
      return
    }

    const success = send({
      action: 'send',
      chat_id: chatId,
      text: text || ' ',
      reply_to_id: replyToId ?? undefined,
    })

    if (!success) {
      showToast('Не удалось отправить сообщение, повторите.', 'error')
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
          <div className="empty-hero__badge">nothing</div>
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
            <button className="btn btn-outline-light btn-sm back-btn" onClick={onBackToList}>
              ←
            </button>
          )}
          <span className="avatar avatar-sm">
            <img src={otherAvatar || '/img/default-avatar.svg'} alt="avatar" />
          </span>
          <div className="chat-header__info">
            <span className="chat-peer">{otherUsername}</span>
            <div className="chat-header__meta">
              <span className="dot online" />
              <span className="chat-subtitle">{isConnected ? 'В сети' : 'Переподключаемся...'}</span>
            </div>
          </div>
        </div>
        <div className="chat-header__actions">
          {!isConnected && <span className="badge bg-warning text-dark">Reconnecting</span>}
          <button className="btn btn-outline-light btn-sm" onClick={onRefreshChats}>
            Обновить список
          </button>
        </div>
      </div>

      <MessageList
        messages={messages}
        currentUserId={currentUserId}
        otherUsername={otherUsername}
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
