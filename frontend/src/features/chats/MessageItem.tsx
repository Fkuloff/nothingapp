import { useCallback, useEffect, useMemo, useState } from 'react'

import { endpoints } from '../../shared/api/endpoints'
import type { Attachment, Message } from '../../shared/api/types'
import { formatFileSize, formatMessageTime } from '../../shared/utils'
import { ImageLightbox } from './ImageLightbox'

// Detect messages containing only 1–3 emojis (sticker-style)
const EMOJI_REGEX = /^(\p{Emoji_Presentation}|\p{Emoji}\uFE0F){1,3}$/u
function isEmojiOnly(text: string): boolean {
  return EMOJI_REGEX.test(text.trim())
}

type Props = {
  message: Message
  isOwn: boolean
  senderName: string
  senderColor?: string
  replyToMessage?: Message | null
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
}

type ContextMenuState = {
  visible: boolean
  x: number
  y: number
}

type AttachmentViewProps = {
  att: Attachment
  onImageClick: (src: string, alt: string) => void
}

function AttachmentView({ att, onImageClick }: AttachmentViewProps) {
  const url = useMemo(() => endpoints.attachments.get(att.id), [att.id])

  if (!att.id) return null

  if (att.file_type === 'image') {
    return (
      <button
        type="button"
        className="attachment-image-btn"
        onClick={() => onImageClick(url, att.file_name)}
      >
        <img src={url} alt={att.file_name} loading="lazy" />
      </button>
    )
  }

  if (att.file_type === 'video') {
    return (
      <video controls>
        <source src={url} type={att.mime_type} />
      </video>
    )
  }

  return (
    <a href={url} download={att.file_name} className="attachment-document">
      <span className="attachment-icon">📄</span>
      <div className="attachment-info">
        <span className="attachment-name">{att.file_name}</span>
        <span className="attachment-size">{formatFileSize(att.file_size)}</span>
      </div>
    </a>
  )
}

export function MessageItem({
  message,
  isOwn,
  senderName,
  senderColor,
  replyToMessage,
  onReply,
  onEdit,
  onDelete,
}: Props) {
  const [contextMenu, setContextMenu] = useState<ContextMenuState>({
    visible: false,
    x: 0,
    y: 0,
  })
  const [lightboxImage, setLightboxImage] = useState<{ src: string; alt: string } | null>(null)

  const handleImageClick = useCallback((src: string, alt: string) => {
    setLightboxImage({ src, alt })
  }, [])

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    if (message.is_deleted) return
    e.preventDefault()

    const menuWidth = 150
    const menuHeight = 160
    const padding = 8

    let x = e.clientX
    let y = e.clientY

    // Prevent menu from going off-screen right
    if (x + menuWidth + padding > window.innerWidth) {
      x = window.innerWidth - menuWidth - padding
    }

    // Prevent menu from going off-screen bottom
    if (y + menuHeight + padding > window.innerHeight) {
      y = window.innerHeight - menuHeight - padding
    }

    setContextMenu({ visible: true, x, y })
  }, [message.is_deleted])

  const closeContextMenu = useCallback(() => {
    setContextMenu((prev) => ({ ...prev, visible: false }))
  }, [])

  const handleReply = useCallback(() => {
    onReply(message.id)
    closeContextMenu()
  }, [message.id, onReply, closeContextMenu])

  const handleEdit = useCallback(() => {
    onEdit(message.id, message.text)
    closeContextMenu()
  }, [message.id, message.text, onEdit, closeContextMenu])

  const handleDelete = useCallback(() => {
    onDelete(message.id)
    closeContextMenu()
  }, [message.id, onDelete, closeContextMenu])

  const handleCopyText = useCallback(() => {
    navigator.clipboard.writeText(message.text)
    closeContextMenu()
  }, [message.text, closeContextMenu])

  // Close context menu on click outside or scroll
  useEffect(() => {
    if (!contextMenu.visible) return

    const handleClickOutside = () => closeContextMenu()
    const handleScroll = () => closeContextMenu()

    document.addEventListener('click', handleClickOutside)
    document.addEventListener('scroll', handleScroll, true)

    return () => {
      document.removeEventListener('click', handleClickOutside)
      document.removeEventListener('scroll', handleScroll, true)
    }
  }, [contextMenu.visible, closeContextMenu])

  const emojiOnly = !message.is_deleted && isEmojiOnly(message.text)

  return (
    <>
      <div
        className={`message ${isOwn ? 'message-self' : 'message-other'}${message.is_deleted ? ' deleted' : ''}${emojiOnly ? ' message--emoji-only' : ''}`}
        data-msg-id={message.id}
        data-user-id={message.user_id}
        onContextMenu={handleContextMenu}
      >
        {replyToMessage && (
          <div className="reply-preview">
            <span className="reply-tag">↩</span>{' '}
            {replyToMessage.is_deleted ? (
              <i>Сообщение удалено</i>
            ) : (
              (replyToMessage.text || '[Пустое сообщение]').substring(0, 64)
            )}
          </div>
        )}

        <div className="message__header">
          <span className="message-sender" style={senderColor ? { color: senderColor } : undefined}>{senderName}</span>
          <span className="message-time">{formatMessageTime(message.created_at)}</span>
        </div>

        <div className="message__content">
          {message.is_deleted ? (
            <span className="message-text text-muted">
              <i>Сообщение удалено</i>
            </span>
          ) : (
            <>
              <span className={`message-text${emojiOnly ? ' message-text--emoji-only' : ''}`}>
                {message.text || '[Пустое сообщение]'}
                {message.edited_at && <span className="edited-indicator"> (ред.)</span>}
              </span>

              {message.attachments && message.attachments.length > 0 && (
                <div className="message-attachments">
                  {message.attachments
                    .filter((att) => att.id)
                    .map((att) => (
                      <div key={att.id} className="attachment-item">
                        <AttachmentView
                          att={att}
                          onImageClick={handleImageClick}
                        />
                      </div>
                    ))}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {contextMenu.visible && (
        <div
          className="context-menu"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="context-menu-item" onClick={handleReply}>
            ↩ Ответить
          </div>
          <div className="context-menu-item" onClick={handleCopyText}>
            📋 Копировать
          </div>
          {isOwn && (
            <>
              <div className="context-menu-item" onClick={handleEdit}>
                ✏️ Редактировать
              </div>
              <div className="context-menu-item danger" onClick={handleDelete}>
                🗑 Удалить
              </div>
            </>
          )}
        </div>
      )}

      {lightboxImage && (
        <ImageLightbox
          src={lightboxImage.src}
          alt={lightboxImage.alt}
          onClose={() => setLightboxImage(null)}
        />
      )}
    </>
  )
}
