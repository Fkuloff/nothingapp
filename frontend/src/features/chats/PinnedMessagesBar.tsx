import { useState } from 'react'

import type { PinnedMessage } from '../../shared/api/types'

type Props = {
  pinnedMessages: PinnedMessage[]
  onScrollToMessage: (messageId: number) => void
  onClose: () => void
}

export function PinnedMessagesBar({ pinnedMessages, onScrollToMessage, onClose }: Props) {
  const [expanded, setExpanded] = useState(false)

  if (pinnedMessages.length === 0) return null

  const scrollTo = (messageId: number) => {
    onScrollToMessage(messageId)
  }

  return (
    <div className="pinned-bar">
      <div className="pinned-bar__header">
        <span className="pinned-bar__icon">📌</span>
        <span className="pinned-bar__count">
          {pinnedMessages.length === 1
            ? 'Закреплённое сообщение'
            : `${pinnedMessages.length} закреплённых`}
        </span>
        {pinnedMessages.length > 1 && (
          <button
            type="button"
            className="pinned-bar__toggle"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? 'Свернуть' : 'Все'}
          </button>
        )}
        <button type="button" className="pinned-bar__close" onClick={onClose} aria-label="Скрыть">
          &times;
        </button>
      </div>

      {!expanded && (
        <button
          type="button"
          className="pinned-bar__item"
          onClick={() => scrollTo(pinnedMessages[0].message_id)}
        >
          <span className="pinned-bar__text">
            {pinnedMessages[0].message.is_deleted
              ? 'Сообщение удалено'
              : (pinnedMessages[0].message.text || '[Вложение]').substring(0, 100)}
          </span>
        </button>
      )}

      {expanded && (
        <div className="pinned-bar__list fancy-scroll">
          {pinnedMessages.map((pin) => (
            <button
              key={pin.id}
              type="button"
              className="pinned-bar__item"
              onClick={() => scrollTo(pin.message_id)}
            >
              <span className="pinned-bar__text">
                {pin.message.is_deleted
                  ? 'Сообщение удалено'
                  : (pin.message.text || '[Вложение]').substring(0, 100)}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
