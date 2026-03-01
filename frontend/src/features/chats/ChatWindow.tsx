import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { endpoints } from '../../shared/api/endpoints'
import { httpPost } from '../../shared/api/httpClient'
import type { GroupMember, Message, PinnedMessage, WSMessageAction } from '../../shared/api/types'
import { PhoneIcon } from '../../shared/components/Icons'
import { useToast } from '../../shared/components/ToastContext'
import { useCallContext } from '../calls/CallContext'
import { UserProfileModal } from '../profile/UserProfileModal'
import { ChatSearch } from './ChatSearch'
import { EmojiPicker } from './EmojiPicker'
import { GroupInfoPanel } from './GroupInfoPanel'
import { MessageComposer } from './MessageComposer'
import { MessageList } from './MessageList'
import { PinnedMessagesBar } from './PinnedMessagesBar'

type Props = {
  chatId?: number
  messages: Message[]
  otherUserId?: number
  otherUsername?: string
  otherAvatar?: string | null
  currentUserId?: number
  loading?: boolean
  error?: string | null
  isConnected: boolean
  isOtherUserOnline?: boolean
  send: (data: WSMessageAction) => boolean
  isMobile?: boolean
  onBackToList?: () => void
  onClearChat?: (chatId: number) => void
  onDeleteChat?: (chatId: number) => void
  // Group props
  isGroup?: boolean
  groupName?: string
  groupMembers?: GroupMember[]
  onGroupUpdated?: () => void
  onGroupDeleted?: () => void
  onGroupLeft?: () => void
  // Pin props
  pinnedMessages?: PinnedMessage[]
  canPin?: boolean
  onPinMessage?: (chatId: number, msgId: number) => void
  onUnpinMessage?: (chatId: number, msgId: number) => void
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
  isConnected,
  isOtherUserOnline = false,
  send,
  isMobile,
  onBackToList,
  onClearChat,
  onDeleteChat,
  isGroup = false,
  groupName = '',
  groupMembers = [],
  onGroupUpdated,
  onGroupDeleted,
  onGroupLeft,
  pinnedMessages = [],
  canPin = false,
  onPinMessage,
  onUnpinMessage,
}: Props) {
  const [messageText, setMessageText] = useState('')
  const [replyToId, setReplyToId] = useState<number | null>(null)
  const [editingMessageId, setEditingMessageId] = useState<number | null>(null)
  const [selectedFiles, setSelectedFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  const [sending, setSending] = useState(false)
  const [isProfileModalOpen, setIsProfileModalOpen] = useState(false)
  const [isGroupInfoOpen, setIsGroupInfoOpen] = useState(false)
  const [isSearchOpen, setIsSearchOpen] = useState(false)
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const [showEmojiPanel, setShowEmojiPanel] = useState(false)
  const [showPinnedBar, setShowPinnedBar] = useState(true)
  const menuRef = useRef<HTMLDivElement>(null)
  const pendingUploadRef = useRef<{ chatId: number; files: File[] } | null>(null)
  const prevMessagesLenRef = useRef(messages.length)

  // Reset input state when switching chats
  useEffect(() => {
    setMessageText('')
    setReplyToId(null)
    setEditingMessageId(null)
    setSelectedFiles([])
    setSending(false)
    setShowEmojiPanel(false)
    setShowPinnedBar(true)
    setIsSearchOpen(false)
    setIsMenuOpen(false)
    setIsGroupInfoOpen(false)
    pendingUploadRef.current = null
  }, [chatId])

  // Close kebab menu on outside click
  useEffect(() => {
    if (!isMenuOpen) return
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [isMenuOpen])

  const handleClearChat = useCallback(() => {
    if (!chatId) return
    if (!confirm('Очистить историю сообщений?')) return
    setIsMenuOpen(false)
    onClearChat?.(chatId)
  }, [chatId, onClearChat])

  const handleDeleteChat = useCallback(() => {
    if (!chatId) return
    if (!confirm('Удалить чат? Это действие нельзя отменить.')) return
    setIsMenuOpen(false)
    onDeleteChat?.(chatId)
  }, [chatId, onDeleteChat])

  const { showToast } = useToast()
  const { callState, initiateCall } = useCallContext()

  const handleStartCall = useCallback(() => {
    if (!chatId || !otherUserId || !otherUsername) return
    initiateCall(chatId, otherUserId, otherUsername, otherAvatar)
  }, [chatId, otherUserId, otherUsername, otherAvatar, initiateCall])

  // Upload files when the sender's new message arrives via WebSocket broadcast
  useEffect(() => {
    const prevLen = prevMessagesLenRef.current
    prevMessagesLenRef.current = messages.length

    // Clear sending state when a new message from current user arrives
    if (messages.length > prevLen) {
      const newMessage = messages[messages.length - 1]
      if (newMessage && newMessage.user_id === currentUserId) {
        setSending(false)
      }
    }

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

      for (const file of files) {
        formData.append('attachments', file, file.name)
      }

      try {
        await httpPost(endpoints.attachments.upload(uploadChatId, messageId), formData)
      } catch (err) {
        console.error('Ошибка загрузки вложений:', err)
        showToast('Не удалось загрузить вложения', 'error')
      } finally {
        setUploading(false)
      }
    }

    doUpload().catch(console.error)
  }, [messages, currentUserId, showToast])

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

    setSending(true)
    const success = send({
      action: 'send',
      chat_id: chatId,
      text: text || ' ',
      reply_to_id: replyToId ?? undefined,
    })

    if (success) {
      setMessageText('')
      setReplyToId(null)
      setSelectedFiles([])
    } else {
      setSending(false)
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

  const handleAddEmoji = useCallback((emoji: string) => {
    setMessageText((prev) => prev + emoji)
  }, [])

  const handleToggleEmoji = useCallback(() => {
    setShowEmojiPanel((prev) => !prev)
  }, [])

  const pinnedMessageIds = useMemo(
    () => new Set(pinnedMessages.map((p) => p.message_id)),
    [pinnedMessages],
  )

  const handlePin = useCallback(
    (msgId: number) => {
      if (chatId) onPinMessage?.(chatId, msgId)
    },
    [chatId, onPinMessage],
  )

  const handleUnpin = useCallback(
    (msgId: number) => {
      if (chatId) onUnpinMessage?.(chatId, msgId)
    },
    [chatId, onUnpinMessage],
  )

  const scrollToMessage = useCallback((messageId: number, highlight = true) => {
    const el = document.getElementById(`msg-${messageId}`)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      if (highlight) {
        el.classList.add('highlight')
        setTimeout(() => el.classList.remove('highlight'), 2000)
      }
    }
  }, [])

  const displayName = isGroup ? groupName : otherUsername
  const displayAvatar = otherAvatar

  if (!chatId || (!displayName && !isGroup)) {
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

  const handleHeaderClick = () => {
    if (isGroup) {
      setIsGroupInfoOpen(true)
    } else {
      setIsProfileModalOpen(true)
    }
  }

  return (
    <div className={`chat-window glassy${showEmojiPanel ? ' chat-window--emoji-open' : ''}`}>
      <div className="chat-window__main">
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
              onClick={handleHeaderClick}
            >
              <span className="avatar avatar-sm">
                <img src={displayAvatar || '/img/default-avatar.svg'} alt="avatar" />
              </span>
              <div className="chat-header__info">
                <span className="chat-peer">{displayName}</span>
                <div className="chat-header__meta">
                  {isGroup ? (
                    <span className="chat-subtitle">{groupMembers.length} участник(ов)</span>
                  ) : (
                    <>
                      <span className={`dot ${isOtherUserOnline ? 'online' : 'offline'}`} />
                      <span className="chat-subtitle">{isOtherUserOnline ? 'В сети' : 'Не в сети'}</span>
                    </>
                  )}
                </div>
              </div>
            </button>
          </div>
          <div className="chat-header__actions">
            {!isGroup && otherUserId && (
              <button
                className="chat-header__call-btn"
                onClick={handleStartCall}
                disabled={callState.status !== 'idle' || !isOtherUserOnline}
                aria-label="Аудиозвонок"
              >
                <PhoneIcon size={20} />
              </button>
            )}
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
            <div className="chat-menu" ref={menuRef}>
              <button
                className="chat-header__menu-btn"
                onClick={() => setIsMenuOpen((v) => !v)}
                aria-label="Меню чата"
              >
                <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
                  <circle cx="12" cy="5" r="1.5" />
                  <circle cx="12" cy="12" r="1.5" />
                  <circle cx="12" cy="19" r="1.5" />
                </svg>
              </button>
              {isMenuOpen && (
                <div className="chat-menu__dropdown">
                  <button className="chat-menu__item" onClick={handleClearChat}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="16" height="16">
                      <path d="M12 2v6M12 22v-6M4.93 4.93l4.24 4.24M14.83 14.83l4.24 4.24M2 12h6M22 12h-6M4.93 19.07l4.24-4.24M14.83 9.17l4.24-4.24" />
                    </svg>
                    Очистить чат
                  </button>
                  {!isGroup && (
                    <button className="chat-menu__item chat-menu__item--danger" onClick={handleDeleteChat}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="16" height="16">
                        <polyline points="3 6 5 6 21 6" />
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                      </svg>
                      Удалить чат
                    </button>
                  )}
                </div>
              )}
            </div>
            {!isConnected && (
              <span className="badge bg-warning text-dark">Reconnecting</span>
            )}
          </div>
        </div>

        {isSearchOpen && (
          <ChatSearch
            chatId={chatId}
            onResultClick={(messageId) => {
              setIsSearchOpen(false)
              scrollToMessage(messageId)
            }}
            onClose={() => setIsSearchOpen(false)}
          />
        )}

        {showPinnedBar && pinnedMessages.length > 0 && (
          <PinnedMessagesBar
            pinnedMessages={pinnedMessages}
            onScrollToMessage={(id) => scrollToMessage(id, false)}
            onClose={() => setShowPinnedBar(false)}
          />
        )}

        <MessageList
          messages={messages}
          currentUserId={currentUserId}
          otherUsername={otherUsername || groupName}
          isGroup={isGroup}
          groupMembers={groupMembers}
          loading={loading}
          error={error}
          onReply={handleReply}
          onEdit={handleEdit}
          onDelete={handleDelete}
          pinnedMessageIds={pinnedMessageIds}
          canPin={canPin}
          onPin={handlePin}
          onUnpin={handleUnpin}
        />

        <MessageComposer
          messages={messages}
          replyToId={replyToId}
          editingMessageId={editingMessageId}
          messageText={messageText}
          selectedFiles={selectedFiles}
          uploading={uploading}
          sending={sending}
          showEmojiPanel={showEmojiPanel}
          onMessageTextChange={setMessageText}
          onSubmit={handleSubmit}
          onFileSelect={handleFileSelect}
          onRemoveFile={handleRemoveFile}
          onCancelDraft={handleCancelDraft}
          onToggleEmoji={handleToggleEmoji}
        />
      </div>

      {showEmojiPanel && (
        <EmojiPicker
          onSelect={handleAddEmoji}
          onClose={() => setShowEmojiPanel(false)}
        />
      )}

      {!isGroup && otherUserId && (
        <UserProfileModal
          isOpen={isProfileModalOpen}
          onClose={() => setIsProfileModalOpen(false)}
          userId={otherUserId}
          username={otherUsername || ''}
          avatarUrl={otherAvatar}
          isOnline={isOtherUserOnline}
        />
      )}

      {isGroup && currentUserId && (
        <GroupInfoPanel
          isOpen={isGroupInfoOpen}
          onClose={() => setIsGroupInfoOpen(false)}
          chatId={chatId}
          groupName={groupName}
          avatarUrl={displayAvatar}
          members={groupMembers}
          currentUserId={currentUserId}
          onGroupUpdated={() => { onGroupUpdated?.() }}
          onGroupDeleted={() => { setIsGroupInfoOpen(false); onGroupDeleted?.() }}
          onGroupLeft={() => { setIsGroupInfoOpen(false); onGroupLeft?.() }}
        />
      )}
    </div>
  )
}
