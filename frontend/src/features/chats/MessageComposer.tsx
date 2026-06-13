import { useEffect, useRef, useState } from 'react'

import type { Message } from '../../shared/api/types'
import { formatFileSize,getFileIcon } from '../../shared/utils'

// Clipboard images (e.g. screenshots) often arrive without a usable filename.
// Synthesize one from the mime type so the attachment shows something sensible
// (the real name is what ends up encrypted in the attachment metadata).
function renameClipboardFile(file: File): File {
  const ext = (file.type.split('/')[1] || 'bin').split(';')[0]
  return new File([file], `pasted-${Date.now()}.${ext}`, { type: file.type || 'application/octet-stream' })
}

type Props = {
  messages: Message[]
  replyToId: number | null
  editingMessageId: number | null
  messageText: string
  selectedFiles: File[]
  uploading: boolean
  sending?: boolean
  showEmojiPanel?: boolean
  /** When true, the input + send button are disabled and styled greyed out.
   *  Used to gate sends behind E2E-readiness: if any recipient hasn't set up
   *  encryption, the composer is blocked until they do. */
  disabled?: boolean
  onMessageTextChange: (text: string) => void
  onSubmit: (event: React.FormEvent) => void
  onFileSelect: (files: File[]) => void
  onRemoveFile: (index: number) => void
  onCancelDraft: () => void
  onToggleEmoji: () => void
  onVoiceRecorded: (file: File, duration: number) => void
}

export function MessageComposer({
  messages,
  replyToId,
  editingMessageId,
  messageText,
  selectedFiles,
  uploading,
  sending,
  showEmojiPanel,
  disabled = false,
  onMessageTextChange,
  onSubmit,
  onFileSelect,
  onRemoveFile,
  onCancelDraft,
  onToggleEmoji,
  onVoiceRecorded,
}: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const messageInputRef = useRef<HTMLTextAreaElement>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<BlobPart[]>([])
  const recordStartedAtRef = useRef(0)
  const pointerStartXRef = useRef(0)
  const lockedRef = useRef(false)
  const cancelledRef = useRef(false)
  const [recording, setRecording] = useState(false)
  const [recordingLocked, setRecordingLocked] = useState(false)
  const [recordingSeconds, setRecordingSeconds] = useState(0)

  const stopMediaStream = () => {
    mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
    mediaStreamRef.current = null
  }

  const preferredRecorderMime = () => {
    const candidates = ['audio/ogg;codecs=opus', 'audio/webm;codecs=opus', 'audio/ogg', 'audio/webm']
    return candidates.find((mime) => MediaRecorder.isTypeSupported(mime)) || ''
  }

  const startRecording = async (clientX: number) => {
    if (disabled || uploading || sending || recording) return
    if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') return
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mimeType = preferredRecorderMime()
      const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined)
      chunksRef.current = []
      pointerStartXRef.current = clientX
      recordStartedAtRef.current = Date.now()
      lockedRef.current = false
      cancelledRef.current = false
      mediaStreamRef.current = stream
      mediaRecorderRef.current = recorder
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) chunksRef.current.push(event.data)
      }
      recorder.onstop = () => {
        stopMediaStream()
        const duration = Math.max(1, Math.round((Date.now() - recordStartedAtRef.current) / 1000))
        const chunks = chunksRef.current
        chunksRef.current = []
        setRecording(false)
        setRecordingLocked(false)
        setRecordingSeconds(0)
        if (cancelledRef.current || chunks.length === 0) return
        const blob = new Blob(chunks, { type: 'audio/ogg' })
        onVoiceRecorded(new File([blob], `voice-${Date.now()}.ogg`, { type: 'audio/ogg' }), duration)
      }
      recorder.start()
      setRecordingSeconds(0)
      setRecording(true)
    } catch (err) {
      console.warn('voice recording failed:', err)
      stopMediaStream()
      setRecording(false)
    }
  }

  const finishRecording = (cancel = false) => {
    const recorder = mediaRecorderRef.current
    if (!recorder || recorder.state === 'inactive') return
    cancelledRef.current = cancel
    recorder.stop()
  }

  useEffect(() => {
    if (!recording) return
    const tick = window.setInterval(() => {
      setRecordingSeconds(Math.floor((Date.now() - recordStartedAtRef.current) / 1000))
    }, 250)
    return () => window.clearInterval(tick)
  }, [recording])

  // Auto-grow vertically with content, capped at ~6 lines (200px). On every
  // value change we measure scrollHeight, which forces a layout — fine for a
  // single textarea, not a hot path. Reset to 'auto' first so the textarea
  // can also *shrink* when text is deleted (otherwise it would stay tall).
  const MAX_HEIGHT = 200
  useEffect(() => {
    const el = messageInputRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, MAX_HEIGHT)}px`
  }, [messageText])

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    if (files.length > 0) {
      onFileSelect(files)
    }
  }

  // Paste images/files (Ctrl+V) straight into the composer — pull any "file"
  // items out of the clipboard and attach them. preventDefault only when we
  // actually consumed files, so pasting plain text still works normally.
  const handlePaste = (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
    if (disabled || uploading) return
    const items = event.clipboardData?.items
    if (!items) return
    const pasted: File[] = []
    for (const item of Array.from(items)) {
      if (item.kind !== 'file') continue
      const file = item.getAsFile()
      if (file) pasted.push(file.name ? file : renameClipboardFile(file))
    }
    if (pasted.length > 0) {
      event.preventDefault()
      onFileSelect(pasted)
    }
  }

  const replyToMessage = replyToId ? messages.find((m) => m.id === replyToId) : null
  // Attachment-only messages have empty text — show a marker so the reply
  // banner isn't blank.
  const replyPreviewText = replyToMessage
    ? replyToMessage.text?.trim()
      ? replyToMessage.text.trim().slice(0, 50)
      : replyToMessage.attachments?.length
        ? '📎 Вложение'
        : ''
    : ''

  const recordingTime = `${Math.floor(recordingSeconds / 60)}:${String(recordingSeconds % 60).padStart(2, '0')}`

  return (
    <form id="chat-form" className="chat-composer" onSubmit={onSubmit}>
      {(replyToId || editingMessageId) && (
        <div className="reply-preview-container">
          <span>
            {editingMessageId
              ? 'Редактирование сообщения'
              : `Ответ на: ${replyPreviewText}`}
          </span>
          <button type="button" className="btn btn-sm btn-secondary" onClick={onCancelDraft}>
            Отмена
          </button>
        </div>
      )}

      {selectedFiles.length > 0 && (
        <div className="attachments-preview">
          {selectedFiles.map((file, index) => (
            <div key={`${file.name}-${index}`} className="attachment-preview-pill">
              <span className="attachment-preview-icon">{getFileIcon(file)}</span>
              <div className="attachment-preview-meta">
                <span className="attachment-preview-name">{file.name}</span>
                <span className="attachment-preview-size">{formatFileSize(file.size)}</span>
              </div>
              <button type="button" className="btn icon-btn remove-attachment" onClick={() => onRemoveFile(index)}>
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
          className="hidden"
          multiple
          onChange={handleFileChange}
        />
        <button
          type="button"
          className="btn icon-btn attach-btn"
          title="Прикрепить файл"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading || disabled}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
          </svg>
        </button>
        {recording ? (
          <div className="voice-recording-bar">
            <span className="voice-recording-dot" />
            <span className="voice-recording-time">{recordingTime}</span>
            <span className="voice-recording-hint">
              {recordingLocked ? 'Запись зафиксирована' : 'Свайп влево - отмена'}
            </span>
            <button type="button" className="btn icon-btn voice-cancel-btn" title="Отмена" onClick={() => finishRecording(true)}>
              ×
            </button>
            <button
              type="button"
              className={`btn icon-btn voice-lock-btn${recordingLocked ? ' voice-lock-btn--active' : ''}`}
              title="Lock"
              onClick={() => {
                lockedRef.current = true
                setRecordingLocked(true)
              }}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="5" y="11" width="14" height="10" rx="2" />
                <path d="M8 11V7a4 4 0 0 1 8 0v4" />
              </svg>
            </button>
          </div>
        ) : (
        <textarea
          id="message"
          ref={messageInputRef}
          className="form-control composer-input"
          placeholder="Напишите сообщение..."
          rows={1}
          value={messageText}
          onChange={(e) => onMessageTextChange(e.target.value)}
          onPaste={handlePaste}
          onKeyDown={(e) => {
            // Enter sends; Shift+Enter inserts a newline (default textarea behavior).
            // Skip the send shortcut on mobile Android Capacitor WebView where
            // physical keyboards aren't the norm and IME composition uses Enter.
            if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing &&
                (messageText.trim() || selectedFiles.length > 0)) {
              e.preventDefault()
              onSubmit(e as unknown as React.FormEvent)
            }
          }}
          disabled={uploading || disabled}
          style={{ resize: 'none', overflowY: 'auto', maxHeight: MAX_HEIGHT, minHeight: 38, lineHeight: '1.4' }}
        />
        )}
        <div className="composer-right">
          <button
            type="button"
            className={`btn icon-btn emoji-toggle-large${showEmojiPanel ? ' emoji-toggle-active' : ''}`}
            title="Смайлики"
            onClick={onToggleEmoji}
            disabled={uploading || disabled}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="12" cy="12" r="10" />
              <path d="M8 14s1.5 2 4 2 4-2 4-2" />
              <line x1="9" y1="9" x2="9.01" y2="9" strokeWidth="3" strokeLinecap="round" />
              <line x1="15" y1="9" x2="15.01" y2="9" strokeWidth="3" strokeLinecap="round" />
            </svg>
          </button>
          <button
            type="submit"
            // Prevent the button from stealing focus on tap — keeps the Android soft keyboard open
            // after send (browser default on mousedown is to shift focus to the button).
            onMouseDown={(e) => e.preventDefault()}
            className={`btn icon-btn send-btn${sending ? ' send-btn--sending' : ''}`}
            title="Отправить"
            disabled={uploading || disabled || sending || (!messageText.trim() && selectedFiles.length === 0)}
          >
            {sending ? (
              <span className="send-spinner" />
            ) : (
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
              </svg>
            )}
          </button>
          <button
            type="button"
            className={`btn icon-btn voice-record-btn${recording ? ' voice-record-btn--active' : ''}`}
            title="Голосовое сообщение"
            disabled={uploading || disabled || sending}
            onPointerDown={(e) => {
              e.preventDefault()
              void startRecording(e.clientX)
            }}
            onPointerMove={(e) => {
              if (!recording || lockedRef.current) return
              if (e.clientX - pointerStartXRef.current < -80) finishRecording(true)
            }}
            onPointerUp={() => {
              if (recording && !lockedRef.current) finishRecording(false)
            }}
            onPointerCancel={() => {
              if (recording && !lockedRef.current) finishRecording(true)
            }}
            onClick={() => {
              if (recording && lockedRef.current) finishRecording(false)
            }}
          >
            {recording && recordingLocked ? (
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z" />
                <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                <path d="M12 19v3" />
              </svg>
            )}
          </button>
        </div>
      </div>
    </form>
  )
}
