import { memo, useCallback, useEffect, useState } from 'react'

import type { Attachment, Message } from '../../shared/api/types'
import { decryptFile, unwrapFileKey } from '../../shared/crypto/e2e'
import { getChatKey } from '../../shared/crypto/peerKeys'
import { formatFileSize, formatMessageTime } from '../../shared/utils'
import { useAccountKey } from '../auth/AccountKey'
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
  isPinned?: boolean
  canPin?: boolean
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
  onPin?: (msgId: number) => void
  onUnpin?: (msgId: number) => void
}

type ContextMenuState = {
  visible: boolean
  x: number
  y: number
}

type AttachmentViewProps = {
  att: Attachment
  senderUserId: number
  onImageClick: (src: string, alt: string) => void
}

/**
 * Renders one attachment. Branches on whether the attachment is E2E
 * (encrypted_file_key + envelope_iv + file_iv all present): if so, fetch the
 * ciphertext blob, unwrap the file_key, decrypt the body, render via blob URL.
 * Otherwise (legacy plaintext attachment, shouldn't exist post-cleanup) render
 * the presigned URL directly.
 */
function AttachmentView({ att, senderUserId, onImageClick }: AttachmentViewProps) {
  const accountKeyCtx = useAccountKey()
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const isEncrypted = Boolean(att.encrypted_file_key && att.envelope_iv && att.file_iv)

  useEffect(() => {
    if (!isEncrypted) return
    if (accountKeyCtx.state.status !== 'ready') return
    if (!att.url) return

    let cancelled = false
    let currentUrl: string | null = null

    const run = async () => {
      try {
        const accountKey = accountKeyCtx.state.status === 'ready' ? accountKeyCtx.state.key : null
        if (!accountKey) throw new Error('account_key missing')
        const chatKey = await getChatKey(accountKey, senderUserId)
        if (!chatKey) throw new Error('cannot derive chat_key for sender')
        const fileKey = await unwrapFileKey(att.encrypted_file_key as string, att.envelope_iv as string, chatKey)
        const ctRes = await fetch(att.url as string)
        if (!ctRes.ok) throw new Error(`fetch ${ctRes.status}`)
        const ctBuf = new Uint8Array(await ctRes.arrayBuffer())
        const decrypted = await decryptFile(ctBuf, att.file_iv as string, fileKey, att.mime_type)
        if (cancelled) return
        currentUrl = URL.createObjectURL(decrypted)
        setBlobUrl(currentUrl)
      } catch (err) {
        if (!cancelled) {
          console.warn('attachment decrypt failed:', err)
          setError(err instanceof Error ? err.message : String(err))
        }
      }
    }
    run()

    return () => {
      cancelled = true
      if (currentUrl) URL.revokeObjectURL(currentUrl)
    }
  }, [
    isEncrypted,
    accountKeyCtx.state,
    senderUserId,
    att.url,
    att.encrypted_file_key,
    att.envelope_iv,
    att.file_iv,
    att.mime_type,
  ])

  if (!att.id || !att.url) return null

  // For E2E: wait until blobUrl is ready (or error).
  const url = isEncrypted ? blobUrl : att.url

  if (isEncrypted && error) {
    return <div className="attachment-document attachment-error">🔒 Не удалось расшифровать вложение</div>
  }
  if (!url) {
    return <div className="attachment-document attachment-loading">🔒 Расшифровка…</div>
  }

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

function MessageItemInner({
  message,
  isOwn,
  senderName,
  senderColor,
  replyToMessage,
  isPinned,
  canPin,
  onReply,
  onEdit,
  onDelete,
  onPin,
  onUnpin,
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
    const menuHeight = 200
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

  const handlePin = useCallback(() => {
    onPin?.(message.id)
    closeContextMenu()
  }, [message.id, onPin, closeContextMenu])

  const handleUnpin = useCallback(() => {
    onUnpin?.(message.id)
    closeContextMenu()
  }, [message.id, onUnpin, closeContextMenu])

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
        id={`msg-${message.id}`}
        className={`message ${isOwn ? 'message-self' : 'message-other'}${message.is_deleted ? ' deleted' : ''}${emojiOnly ? ' message--emoji-only' : ''}`}
        data-msg-id={message.id}
        data-user-id={message.user_id}
        onContextMenu={handleContextMenu}
      >
        {isPinned && (
          <span className="message-pin-indicator" title="Закреплено">📌</span>
        )}
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
              {(message.text.trim() || !(message.attachments && message.attachments.length > 0)) && (
                <span className={`message-text${emojiOnly ? ' message-text--emoji-only' : ''}`}>
                  {message.text.trim() || '[Пустое сообщение]'}
                  {message.edited_at && <span className="edited-indicator"> (ред.)</span>}
                </span>
              )}

              {message.attachments && message.attachments.length > 0 && (
                <div className="message-attachments">
                  {message.attachments
                    .filter((att) => att.id)
                    .map((att) => (
                      <div key={att.id} className="attachment-item">
                        <AttachmentView
                          att={att}
                          senderUserId={message.user_id}
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
          {canPin && !isPinned && (
            <div className="context-menu-item" onClick={handlePin}>
              📌 Закрепить
            </div>
          )}
          {canPin && isPinned && (
            <div className="context-menu-item" onClick={handleUnpin}>
              📌 Открепить
            </div>
          )}
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

// Memoised export — a chat with 500+ messages re-renders every bubble on every parent
// state update (presence tick, new incoming message, etc.). With stable callbacks from
// ChatWindow, the default shallow prop compare lets React skip N-1 renders.
export const MessageItem = memo(MessageItemInner)
