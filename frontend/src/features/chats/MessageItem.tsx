import { memo, useCallback, useEffect, useState } from 'react'

import type { Attachment, Message } from '../../shared/api/types'
import { decryptFile, decryptMetadata, unwrapFileKey } from '../../shared/crypto/e2e'
import { getChatKey } from '../../shared/crypto/peerKeys'
import { downloadAttachmentNative } from '../../shared/downloadAttachment'
import { isNative } from '../../shared/platform'
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

// Derive the UI render bucket (image / video / document) from a mime type.
// Used to live server-side as `determineFileType`; moved client-side because
// the server no longer sees the real mime (it's encrypted under file_key
// alongside the body).
function bucketFromMime(mime: string): 'image' | 'video' | 'document' {
  if (mime.startsWith('image/')) return 'image'
  if (mime.startsWith('video/')) return 'video'
  return 'document'
}

/**
 * Renders one attachment. Every attachment is E2E: fetch the ciphertext blob,
 * unwrap the file_key, decrypt both the small metadata blob (filename + mime)
 * and the body, render via blob URL. Rows missing any E2E field
 * (encrypted_file_key, envelope_iv, file_iv, encrypted_metadata, metadata_iv)
 * are treated as broken and surface the "🔒 Не удалось расшифровать"
 * placeholder — legacy plaintext attachments don't exist after the metadata
 * migration.
 */
function AttachmentView({ att, senderUserId, onImageClick }: AttachmentViewProps) {
  const accountKeyCtx = useAccountKey()
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [decryptedBlob, setDecryptedBlob] = useState<Blob | null>(null)
  const [error, setError] = useState<string | null>(null)
  // Decrypted filename + mime — empty until decrypt completes.
  const [decryptedMeta, setDecryptedMeta] = useState<{ fileName: string; mimeType: string } | null>(null)

  const hasAllE2EFields = Boolean(
    att.encrypted_file_key &&
      att.envelope_iv &&
      att.file_iv &&
      att.encrypted_metadata &&
      att.metadata_iv,
  )

  useEffect(() => {
    if (!hasAllE2EFields) return
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

        // Decrypt the metadata blob FIRST so we know the real mime to pass
        // to the Blob constructor (browser sniffs bytes anyway for <img>,
        // but a correct type matters for <video>/download).
        const meta = await decryptMetadata(att.encrypted_metadata as string, att.metadata_iv as string, fileKey)
        if (cancelled) return
        const mime = meta.mimeType || 'application/octet-stream'
        setDecryptedMeta({ fileName: meta.fileName, mimeType: mime })

        const ctRes = await fetch(att.url as string)
        if (!ctRes.ok) throw new Error(`fetch ${ctRes.status}`)
        const ctBuf = new Uint8Array(await ctRes.arrayBuffer())
        const decrypted = await decryptFile(ctBuf, att.file_iv as string, fileKey, mime)
        if (cancelled) return
        currentUrl = URL.createObjectURL(decrypted)
        setBlobUrl(currentUrl)
        // Keep the raw Blob too — native download path needs the bytes
        // (can't read them back from blob: URL through the Capacitor
        // WebView's restricted fetch on a fresh tab).
        setDecryptedBlob(decrypted)
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
    hasAllE2EFields,
    accountKeyCtx.state,
    senderUserId,
    att.url,
    att.encrypted_file_key,
    att.envelope_iv,
    att.file_iv,
    att.encrypted_metadata,
    att.metadata_iv,
  ])

  if (!att.id || !att.url) return null

  // Rows missing any of the five required E2E fields are unrenderable.
  // After the legacy metadata removal, every valid attachment must have
  // all of them — show a clear "not decryptable" placeholder instead of
  // silently rendering a broken file.
  if (!hasAllE2EFields) {
    return <div className="attachment-document attachment-error">🔒 Не удалось расшифровать вложение</div>
  }
  if (error) {
    return <div className="attachment-document attachment-error">🔒 Не удалось расшифровать вложение</div>
  }
  if (!blobUrl) {
    return <div className="attachment-document attachment-loading">🔒 Расшифровка…</div>
  }

  const fileName = decryptedMeta?.fileName || 'Файл'
  const mimeType = decryptedMeta?.mimeType || 'application/octet-stream'
  const bucket = bucketFromMime(mimeType)

  // Native (Capacitor WebView) doesn't honour the `<a download>` attribute —
  // tapping the link does nothing. Intercept here: write the decrypted blob
  // to the app's cache directory and pop the native share sheet so the user
  // can save it to Files / open with another app. On web we fall through
  // and let the browser's download manager handle it as before.
  const handleDocClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (!isNative() || !decryptedBlob) return
    e.preventDefault()
    void downloadAttachmentNative(decryptedBlob, fileName).catch((err) => {
      console.warn('native download failed:', err)
    })
  }

  if (bucket === 'image') {
    return (
      <button
        type="button"
        className="attachment-image-btn"
        onClick={() => onImageClick(blobUrl, fileName)}
      >
        <img src={blobUrl} alt={fileName} loading="lazy" />
      </button>
    )
  }

  if (bucket === 'video') {
    return (
      <video controls>
        <source src={blobUrl} type={mimeType} />
      </video>
    )
  }

  return (
    <a href={blobUrl} download={fileName} className="attachment-document" onClick={handleDocClick}>
      <span className="attachment-icon">📄</span>
      <div className="attachment-info">
        <span className="attachment-name">{fileName}</span>
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
