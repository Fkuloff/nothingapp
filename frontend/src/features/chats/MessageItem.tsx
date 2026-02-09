import type { Message, Attachment } from '../../shared/api/types'
import { endpoints } from '../../shared/api/endpoints'
import { formatMessageTime, formatFileSize } from '../../shared/utils'

type Props = {
  message: Message
  isOwn: boolean
  senderName: string
  replyToMessage?: Message | null
  onReply: (msgId: number) => void
  onEdit: (msgId: number, text: string) => void
  onDelete: (msgId: number) => void
}

function AttachmentView({ att }: { att: Attachment }) {
  // Skip rendering if attachment has no valid id
  if (!att.id) return null

  if (att.file_type === 'image') {
    return (
      <a href={endpoints.attachments.get(att.id)} target="_blank" rel="noopener noreferrer">
        <img src={endpoints.attachments.thumbnail(att.id)} alt={att.file_name} />
      </a>
    )
  }

  if (att.file_type === 'video') {
    return (
      <video controls>
        <source src={endpoints.attachments.get(att.id)} type={att.mime_type} />
      </video>
    )
  }

  return (
    <a href={endpoints.attachments.get(att.id)} download={att.file_name} className="attachment-document">
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
  replyToMessage,
  onReply,
  onEdit,
  onDelete,
}: Props) {
  return (
    <div
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
                      <AttachmentView att={att} />
                    </div>
                  ))}
              </div>
            )}
          </>
        )}
      </div>

      {!message.is_deleted && (
        <div className="message__actions">
          <button className="message-action" type="button" onClick={() => onReply(message.id)}>
            Ответить
          </button>
          {isOwn && (
            <>
              <button className="message-action" type="button" onClick={() => onEdit(message.id, message.text)}>
                Редактировать
              </button>
              <button className="message-action danger" type="button" onClick={() => onDelete(message.id)}>
                Удалить
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}
