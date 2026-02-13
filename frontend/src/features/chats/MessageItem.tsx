import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import type { Message, Attachment } from '../../shared/api/types'
import { endpoints } from '../../shared/api/endpoints'
import { getAuthToken } from '../../shared/api/httpClient'
import { formatMessageTime, formatFileSize } from '../../shared/utils'
import { ImageLightbox } from './ImageLightbox'
import { decryptFile } from '../../shared/crypto/encryption'
import { getOrDeriveChatKey } from '../../shared/crypto/keyExchange'

type Props = {
  message: Message
  isOwn: boolean
  senderName: string
  replyToMessage?: Message | null
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
  chatId?: number
  otherUserId?: number
}

type ContextMenuState = {
  visible: boolean
  x: number
  y: number
}

type AttachmentViewProps = {
  att: Attachment
  chatId?: number
  otherUserId?: number
  onImageClick: (src: string, alt: string) => void
}

/**
 * Fetches and decrypts an encrypted attachment, returning an Object URL.
 * For unencrypted attachments, returns the direct API URL.
 */
function useDecryptedUrl(att: Attachment, chatId?: number, otherUserId?: number) {
  const isEncrypted = Boolean(att.iv && att.id)
  const directUrl = useMemo(
    () => (isEncrypted ? null : endpoints.attachments.get(att.id)),
    [att.id, isEncrypted],
  )

  const [decryptedUrl, setDecryptedUrl] = useState<string | null>(null)
  const [decryptError, setDecryptError] = useState(false)
  const urlRef = useRef<string | null>(null)

  useEffect(() => {
    if (!isEncrypted || !chatId || !otherUserId) return

    let cancelled = false

    const decrypt = async () => {
      const key = await getOrDeriveChatKey(chatId, otherUserId)
      if (!key || cancelled) return

      const BASE_URL = import.meta.env.VITE_USE_PROXY === 'true'
        ? ''
        : import.meta.env.VITE_API_BASE_URL ?? ''
      const token = getAuthToken()
      const headers: HeadersInit = token ? { Authorization: `Bearer ${token}` } : {}

      const response = await fetch(`${BASE_URL}${endpoints.attachments.get(att.id)}`, {
        credentials: 'include',
        headers,
      })
      if (!response.ok || cancelled) return

      const encryptedData = await response.arrayBuffer()
      if (cancelled) return

      const decrypted = await decryptFile(encryptedData, att.iv!, key)
      if (cancelled) return

      const mimeType = att.original_type || att.mime_type
      const blob = new Blob([decrypted], { type: mimeType })
      const blobUrl = URL.createObjectURL(blob)
      urlRef.current = blobUrl
      setDecryptedUrl(blobUrl)
    }

    decrypt().catch(() => {
      if (!cancelled) setDecryptError(true)
    })

    return () => {
      cancelled = true
      if (urlRef.current) {
        URL.revokeObjectURL(urlRef.current)
        urlRef.current = null
      }
    }
  }, [att.id, att.iv, att.mime_type, att.original_type, chatId, otherUserId, isEncrypted])

  const loading = isEncrypted && !decryptedUrl && !decryptError
  return { url: isEncrypted ? decryptedUrl : directUrl, loading }
}

function AttachmentView({ att, chatId, otherUserId, onImageClick }: AttachmentViewProps) {
  const { url, loading } = useDecryptedUrl(att, chatId, otherUserId)

  if (!att.id) return null

  // Use original metadata for encrypted attachments
  const displayName = att.original_name || att.file_name
  const displayType = att.original_type || att.mime_type
  const fileType = att.original_type
    ? (att.original_type.startsWith('image/') ? 'image' : att.original_type.startsWith('video/') ? 'video' : 'document')
    : att.file_type

  if (loading || !url) {
    return <div className="attachment-loading">Расшифровка...</div>
  }

  if (fileType === 'image') {
    return (
      <button
        type="button"
        className="attachment-image-btn"
        onClick={() => onImageClick(url, displayName)}
      >
        <img src={url} alt={displayName} loading="lazy" />
      </button>
    )
  }

  if (fileType === 'video') {
    return (
      <video controls>
        <source src={url} type={displayType} />
      </video>
    )
  }

  return (
    <a href={url} download={displayName} className="attachment-document">
      <span className="attachment-icon">📄</span>
      <div className="attachment-info">
        <span className="attachment-name">{displayName}</span>
        <span className="attachment-size">{formatFileSize(att.file_size)}</span>
      </div>
    </a>
  )
}

export function MessageItem({
  message,
  isOwn,
  senderName,
  replyToMessage,
  onReply,
  onEdit,
  onDelete,
  chatId,
  otherUserId,
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

  return (
    <>
      <div
        className={`message ${isOwn ? 'message-self' : 'message-other'} ${message.is_deleted ? 'deleted' : ''}`}
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
          <span className="message-sender">{senderName}</span>
          <span className="message-time">{formatMessageTime(message.created_at)}</span>
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
                  {message.attachments
                    .filter((att) => att.id)
                    .map((att) => (
                      <div key={att.id} className="attachment-item">
                        <AttachmentView
                          att={att}
                          chatId={chatId}
                          otherUserId={otherUserId}
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
