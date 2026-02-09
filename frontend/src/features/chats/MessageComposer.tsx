import { useRef, useState } from 'react'
import type { Message } from '../../shared/api/types'
import { EmojiPicker } from './EmojiPicker'
import { getFileIcon, formatFileSize } from '../../shared/utils'

type Props = {
  messages: Message[]
  replyToId: number | null
  editingMessageId: number | null
  messageText: string
  selectedFiles: File[]
  uploading: boolean
  onMessageTextChange: (text: string) => void
  onSubmit: (event: React.FormEvent) => void
  onFileSelect: (files: File[]) => void
  onRemoveFile: (index: number) => void
  onCancelDraft: () => void
}

export function MessageComposer({
  messages,
  replyToId,
  editingMessageId,
  messageText,
  selectedFiles,
  uploading,
  onMessageTextChange,
  onSubmit,
  onFileSelect,
  onRemoveFile,
  onCancelDraft,
}: Props) {
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const messageInputRef = useRef<HTMLInputElement>(null)
  const emojiToggleRef = useRef<HTMLButtonElement>(null)

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    if (files.length > 0) {
      onFileSelect(files)
    }
  }

  const handleAddEmoji = (emoji: string) => {
    const input = messageInputRef.current
    const start = input?.selectionStart ?? messageText.length
    const end = input?.selectionEnd ?? messageText.length
    const next = messageText.slice(0, start) + emoji + messageText.slice(end)
    onMessageTextChange(next)

    requestAnimationFrame(() => {
      const pos = start + emoji.length
      if (input) {
        input.focus()
        input.setSelectionRange(pos, pos)
      }
    })
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
            <EmojiPicker
              onSelect={handleAddEmoji}
              onClose={() => setShowEmojiPicker(false)}
              toggleRef={emojiToggleRef}
            />
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
          onChange={(e) => onMessageTextChange(e.target.value)}
          disabled={uploading}
        />
        <button type="submit" className="btn btn-send" disabled={uploading || (!messageText.trim() && selectedFiles.length === 0)}>
          {uploading ? 'Загрузка...' : editingMessageId ? 'Сохранить' : 'Отправить'}
        </button>
      </div>
    </form>
  )
}
