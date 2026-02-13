import { useEffect, useRef, useState } from 'react'
import type { Message, WSMessageAction } from '../../shared/api/types'
import { httpPost } from '../../shared/api/httpClient'
import { endpoints } from '../../shared/api/endpoints'
import { MessageList } from './MessageList'
import { MessageComposer } from './MessageComposer'
import { useToast } from '../../shared/components/ToastContext'
import { UserProfileModal } from '../profile/UserProfileModal'
import { ChatSearch } from './ChatSearch'
import { useChatEncryption } from '../../shared/hooks/useChatEncryption'

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
  const [isProfileModalOpen, setIsProfileModalOpen] = useState(false)
  const [isSearchOpen, setIsSearchOpen] = useState(false)
  const pendingUploadRef = useRef<{ chatId: number; files: File[] } | null>(null)
  const prevMessagesLenRef = useRef(messages.length)

  const { showToast } = useToast()
  const { encryptAction, encryptFileForUpload } = useChatEncryption(otherUserId)

  // Upload files when the sender's new message arrives via WebSocket broadcast
  useEffect(() => {
    const prevLen = prevMessagesLenRef.current
    prevMessagesLenRef.current = messages.length

    if (!pendingUploadRef.current || messages.length <= prevLen) return

    // Find the newest message from the current user
    const newMessage = messages[messages.length - 1]
    if (!newMessage || newMessage.user_id !== currentUserId) return

    const { chatId: uploadChatId, files } = pendingUploadRef.current
    if (newMessage.chat_id !== uploadChatId || files.length === 0) return

    pendingUploadRef.current = null
    const messageId = newMessage.id

    const doUpload = async () => {
      setUploading(true)
      const formData = new FormData()
      const fileIVs: Record<string, string> = {}
      const originalTypes: Record<string, string> = {}
      const originalNames: Record<string, string> = {}

      for (const file of files) {
        const encrypted = await encryptFileForUpload(file, uploadChatId)
        if (encrypted) {
          const encFileName = file.name + '.enc'
          formData.append('attachments', encrypted.blob, encFileName)
          fileIVs[encFileName] = encrypted.iv
          originalTypes[encFileName] = encrypted.originalType
          originalNames[encFileName] = encrypted.originalName
        } else {
          formData.append('attachments', file)
        }
      }

      if (Object.keys(fileIVs).length > 0) {
        formData.append('file_ivs', JSON.stringify(fileIVs))
        formData.append('original_types', JSON.stringify(originalTypes))
        formData.append('original_names', JSON.stringify(originalNames))
      }

      try {
        await httpPost(endpoints.attachments.upload(uploadChatId, messageId), formData)
        onMessagesUpdate?.()
      } catch (err) {
        console.error('Ошибка загрузки вложений:', err)
        showToast('Не удалось загрузить вложения', 'error')
      } finally {
        setUploading(false)
      }
    }

    doUpload().catch(console.error)
  }, [messages, currentUserId, encryptFileForUpload, onMessagesUpdate, showToast])

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
      const editAction = await encryptAction({ action: 'edit', chat_id: chatId, message_id: editingMessageId, text })
      const success = send(editAction)
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

    const sendAction = await encryptAction({
      action: 'send',
      chat_id: chatId,
      text: text || ' ',
      reply_to_id: replyToId ?? undefined,
    })
    const success = send(sendAction)

    if (success) {
      setMessageText('')
      setReplyToId(null)
      setSelectedFiles([])
      // Files will be uploaded by the useEffect when the WS broadcast arrives
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

  if (!chatId || !otherUsername) {
    return (
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
    )
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
          <button
            type="button"
            className="chat-header__link"
            onClick={() => setIsProfileModalOpen(true)}
          >
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
          </button>
        </div>
        <div className="chat-header__actions">
          <button
            className="chat-header__search-btn"
            onClick={() => setIsSearchOpen(!isSearchOpen)}
            aria-label="Поиск по сообщениям"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="20" height="20">
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.35-4.35" />
            </svg>
          </button>
          {!isConnected && (
            <span className="badge bg-warning text-dark">Reconnecting</span>
          )}
        </div>
      </div>

      {isSearchOpen && otherUserId && (
        <ChatSearch
          chatId={chatId}
          otherUserId={otherUserId}
          onResultClick={(messageId) => {
            setIsSearchOpen(false)
            // Scroll to message — the MessageList component should handle this
            const el = document.getElementById(`msg-${messageId}`)
            if (el) {
              el.scrollIntoView({ behavior: 'smooth', block: 'center' })
              el.classList.add('highlight')
              setTimeout(() => el.classList.remove('highlight'), 2000)
            }
          }}
          onClose={() => setIsSearchOpen(false)}
        />
      )}

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

      {otherUserId && (
        <UserProfileModal
          isOpen={isProfileModalOpen}
          onClose={() => setIsProfileModalOpen(false)}
          userId={otherUserId}
          username={otherUsername}
          avatarUrl={otherAvatar}
          isOnline={isOtherUserOnline}
        />
      )}
    </div>
  )
}
