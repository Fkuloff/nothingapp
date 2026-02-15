import { useRef } from 'react'

import type { Message } from '../../shared/api/types'
import { formatFileSize,getFileIcon } from '../../shared/utils'

type Props = {
  messages: Message[]
  replyToId: number | null
  editingMessageId: number | null
  messageText: string
  selectedFiles: File[]
  uploading: boolean
  sending?: boolean
  showEmojiPanel?: boolean
  onMessageTextChange: (text: string) => void
  onSubmit: (event: React.FormEvent) => void
  onFileSelect: (files: File[]) => void
  onRemoveFile: (index: number) => void
  onCancelDraft: () => void
  onToggleEmoji: () => void
  onAddEmoji?: (emoji: string) => void
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
  onMessageTextChange,
  onSubmit,
  onFileSelect,
  onRemoveFile,
  onCancelDraft,
  onToggleEmoji,
}: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const messageInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    if (files.length > 0) {
      onFileSelect(files)
    }
  }

  const replyToMessage = replyToId ? messages.find((m) => m.id === replyToId) : null

  return (
    <form id="chat-form" className="chat-composer" onSubmit={onSubmit}>
      {(replyToId || editingMessageId) && (
        <div className="reply-preview-container">
          <span>
            {editingMessageId
              ? 'Редактирование сообщения'
              : `Ответ на: ${replyToMessage?.text?.substring(0, 50) ?? ''}`}
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
          disabled={uploading}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
          </svg>
        </button>
        <input
          type="text"
          id="message"
          ref={messageInputRef}
          className="form-control composer-input"
          placeholder="Напишите сообщение..."
          value={messageText}
          onChange={(e) => onMessageTextChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && (messageText.trim() || selectedFiles.length > 0)) {
              e.preventDefault()
              onSubmit(e as unknown as React.FormEvent)
            }
          }}
          disabled={uploading}
        />
        <div className="composer-right">
          <button
            type="button"
            className={`btn icon-btn emoji-toggle-large${showEmojiPanel ? ' emoji-toggle-active' : ''}`}
            title="Смайлики"
            onClick={onToggleEmoji}
            disabled={uploading}
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
            className={`btn icon-btn send-btn${sending ? ' send-btn--sending' : ''}`}
            title="Отправить"
            disabled={uploading || sending || (!messageText.trim() && selectedFiles.length === 0)}
          >
            {sending ? (
              <span className="send-spinner" />
            ) : (
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
              </svg>
            )}
          </button>
        </div>
      </div>
    </form>
  )
}
