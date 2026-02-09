import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Message, WSEvent } from '../../shared/api/types'
import { useWebSocket } from '../../shared/hooks/useWebSocket'
import { httpPost } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'

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
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const messageInputRef = useRef<HTMLInputElement>(null)
  const emojiPopoverRef = useRef<HTMLDivElement>(null)
  const emojiToggleRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    setMessages(initialMessages)
  }, [initialMessages])

  const scrollToBottom = useCallback((smooth = false) => {
    messagesEndRef.current?.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto' })
  }, [])

  useEffect(() => {
    scrollToBottom(true)
  }, [messages, scrollToBottom])

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
                alert('Не удалось загрузить вложения')
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
    [chatId, currentUserId, editingMessageId, onMessagesUpdate, onRefreshChats, selectedFiles]
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
      alert('Выберите чат, чтобы отправлять сообщения')
      return
    }

    if (!text && selectedFiles.length === 0) {
      alert('Введите сообщение или прикрепите файлы')
      return
    }

    if (!isConnected) {
      alert('Соединение потеряно, ждём переподключения...')
      return
    }

    if (editingMessageId) {
      const success = send({ action: 'edit', message_id: editingMessageId, text })
      if (!success) {
        alert('Не удалось отправить изменение, повторите.')
      }
      return
    }

    const success = send({
      action: 'send',
      text: text || ' ',
      reply_to_id: replyToId ?? undefined,
    })

    if (!success) {
      alert('Не удалось отправить сообщение, повторите.')
    }
  }

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    if (files.length === 0) return
    setSelectedFiles((prev) => [...prev, ...files])
  }

  const handleAddEmoji = (emoji: string) => {
    const input = messageInputRef.current
    const start = input?.selectionStart ?? messageText.length
    const end = input?.selectionEnd ?? messageText.length
    const next = messageText.slice(0, start) + emoji + messageText.slice(end)
    setMessageText(next)

    requestAnimationFrame(() => {
      const pos = start + emoji.length
      if (input) {
        input.focus()
        input.setSelectionRange(pos, pos)
      }
    })
  }

  useEffect(() => {
    if (!showEmojiPicker) return

    const onClickOutside = (event: MouseEvent) => {
      const popover = emojiPopoverRef.current
      const toggle = emojiToggleRef.current
      if (!popover || !toggle) return

      if (popover.contains(event.target as Node) || toggle.contains(event.target as Node)) {
        return
      }
      setShowEmojiPicker(false)
    }

    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [showEmojiPicker])

  const removeFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, idx) => idx !== index))
  }

  const startReply = (msgId: number) => {
    setEditingMessageId(null)
    setReplyToId(msgId)
    setMessageText('')
  }

  const startEdit = (msgId: number, text: string) => {
    setReplyToId(null)
    setEditingMessageId(msgId)
    setMessageText(text)
  }

  const getSelectedFileIcon = (file: File) => {
    if (file.type.startsWith('image/')) return '🖼️'
    if (file.type.startsWith('video/')) return '🎬'
    if (file.type.startsWith('audio/')) return '🎵'
    if (file.type === 'application/pdf') return '📕'
    return '📎'
  }

  const deleteMessage = (msgId: number) => {
    if (!confirm('Удалить сообщение?')) return
    send({ action: 'delete', message_id: msgId })
  }

  const cancelDraftState = () => {
    setReplyToId(null)
    setEditingMessageId(null)
    setMessageText('')
  }

  const emptyState = useMemo(
    () => (
      <div className="chat-window glassy empty-chat-panel">
        <div className="empty-hero">
          <div className="empty-hero__badge">Messenger</div>
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
          <img className="avatar avatar-sm" src={otherAvatar ?? '/static/img/default-avatar.svg'} alt="avatar" />
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

      <div id="messages" className="chat-body fancy-scroll">
        {error && <div className="alert alert-danger m-3">{error}</div>}
        {loading ? (
          <div className="text-center p-4 text-muted">Загружаем сообщения...</div>
        ) : messages.length === 0 ? (
          <div className="text-center text-muted p-4">В этом чате пока нет сообщений.</div>
        ) : (
          <>
            {messages.map((message) => {
              const isOwn = message.user_id === currentUserId
              const replyToMessage = message.reply_to_id
                ? messages.find((m) => m.id === message.reply_to_id)
                : null

              return (
                <div
                  key={message.id}
                  className={`message ${isOwn ? 'message-self' : 'message-other'} ${message.is_deleted ? 'deleted' : ''}`}
                  data-msg-id={message.id}
                  data-user-id={message.user_id}
                >
                  {replyToMessage && (
                    <div className="reply-preview">
                      <span className="reply-tag">Ответ на:</span>{' '}
                      {replyToMessage.is_deleted ? (
                        <i>Сообщение удалено</i>
                      ) : (
                        (replyToMessage.text || '[Пустое сообщение]').substring(0, 64)
                      )}
                    </div>
                  )}

                  <div className="message__header">
                    <span className="message-sender">{isOwn ? 'Вы' : otherUsername}</span>
                    <span className="message-time">
                      {new Date(message.created_at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </div>

                  <div className="message__content">
                    {message.is_deleted ? (
                      <span className="message-text text-muted">
                        <i>Сообщение удалено</i>
                      </span>
                    ) : (
                      <>
                        <span className="message-text">
                          {message.text || '[Пустое сообщение]'}
                          {message.edited_at && <span className="edited-indicator"> (ред.)</span>}
                        </span>

                        {message.attachments && message.attachments.length > 0 && (
                          <div className="message-attachments">
                            {message.attachments.map((att) => (
                              <div key={att.id} className="attachment-item">
                                {att.file_type === 'image' ? (
                                  <a href={endpoints.attachments.get(att.id)} target="_blank" rel="noopener noreferrer">
                                    <img src={endpoints.attachments.thumbnail(att.id)} alt={att.file_name} />
                                  </a>
                                ) : att.file_type === 'video' ? (
                                  <video controls>
                                    <source src={endpoints.attachments.get(att.id)} type={att.mime_type} />
                                  </video>
                                ) : (
                                  <a href={endpoints.attachments.get(att.id)} download={att.file_name} className="attachment-document">
                                    <span className="attachment-icon">📄</span>
                                    <div className="attachment-info">
                                      <span className="attachment-name">{att.file_name}</span>
                                      <span className="attachment-size">{(att.file_size / 1024).toFixed(1)} KB</span>
                                    </div>
                                  </a>
                                )}
                              </div>
                            ))}
                          </div>
                        )}
                      </>
                    )}
                  </div>

                  {!message.is_deleted && (
                    <div className="message__actions">
                      <button className="message-action" type="button" onClick={() => startReply(message.id)}>
                        Ответить
                      </button>
                      {isOwn && (
                        <>
                          <button className="message-action" type="button" onClick={() => startEdit(message.id, message.text)}>
                            Редактировать
                          </button>
                          <button className="message-action danger" type="button" onClick={() => deleteMessage(message.id)}>
                            Удалить
                          </button>
                        </>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>

      <form id="chat-form" className="chat-composer" onSubmit={handleSubmit}>
        {(replyToId || editingMessageId) && (
          <div className="reply-preview-container">
            <span>
              {editingMessageId
                ? 'Редактирование сообщения'
                : `Ответ на: ${messages.find((m) => m.id === replyToId)?.text?.substring(0, 50) ?? ''}`}
            </span>
            <button type="button" className="btn btn-sm btn-secondary" onClick={cancelDraftState}>
              Отмена
            </button>
          </div>
        )}

        {selectedFiles.length > 0 && (
          <div className="attachments-preview">
            {selectedFiles.map((file, index) => (
              <div key={`${file.name}-${index}`} className="attachment-preview-pill">
                <span className="attachment-preview-icon">{getSelectedFileIcon(file)}</span>
                <div className="attachment-preview-meta">
                  <span className="attachment-preview-name">{file.name}</span>
                  <span className="attachment-preview-size">{(file.size / 1024).toFixed(1)} KB</span>
                </div>
                <button type="button" className="btn icon-btn remove-attachment" onClick={() => removeFile(index)}>
                  ×
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="chat-composer__controls">
          <input
            type="file"
            ref={fileInputRef}
            style={{ display: 'none' }}
            multiple
            onChange={handleFileSelect}
          />
          <div className="composer-left">
            <button
              type="button"
              className="btn icon-btn emoji-toggle"
              title="Смайлики"
              onClick={() => setShowEmojiPicker((prev) => !prev)}
              disabled={uploading}
              ref={emojiToggleRef}
            >
              😊
            </button>
            {showEmojiPicker && (
              <div className="emoji-popover glassy" ref={emojiPopoverRef}>
                {['😀','😁','😂','😉','😍','😎','🤔','😭','🔥','💯','👍','🙏'].map((emoji) => (
                  <button
                    key={emoji}
                    type="button"
                    className="emoji-option"
                    onClick={() => handleAddEmoji(emoji)}
                  >
                    {emoji}
                  </button>
                ))}
              </div>
            )}
            <button
              type="button"
              className="btn icon-btn attach-btn"
              title="Прикрепить файл"
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
            >
              📎
            </button>
          </div>
          <input
            type="text"
            id="message"
            ref={messageInputRef}
            className="form-control composer-input"
            placeholder="Напишите сообщение..."
            value={messageText}
            onChange={(e) => setMessageText(e.target.value)}
            disabled={uploading}
          />
          <button type="submit" className="btn btn-send" disabled={uploading || (!messageText.trim() && selectedFiles.length === 0)}>
            {uploading ? 'Загрузка...' : editingMessageId ? 'Сохранить' : 'Отправить'}
          </button>
        </div>
      </form>
    </div>
  )
}
