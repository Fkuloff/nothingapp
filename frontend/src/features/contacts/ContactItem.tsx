import { useState } from 'react'
import { Link } from 'react-router-dom'

import { ChatBubbleIcon, CloseIcon } from '../../shared/components/Icons'
import { useConfirmAction } from '../../shared/hooks/useConfirmAction'

type Props = {
  id: number
  username: string
  name: string
  avatar_url?: string
  onStartChat: (userId: number) => Promise<void>
  onRemove: (userId: number) => void
}

export function ContactItem({ id, username, name, avatar_url, onStartChat, onRemove }: Props) {
  const [starting, setStarting] = useState(false)
  const [removing, setRemoving] = useState(false)
  const { confirming, startConfirm, cancelConfirm } = useConfirmAction()

  const handleStartChat = async (e: React.MouseEvent) => {
    e.stopPropagation()
    setStarting(true)
    try {
      await onStartChat(id)
    } finally {
      setStarting(false)
    }
  }

  const handleConfirmRemove = async (e: React.MouseEvent) => {
    e.stopPropagation()
    setRemoving(true)
    try {
      onRemove(id)
    } finally {
      setRemoving(false)
      cancelConfirm()
    }
  }

  return (
    <li className={`contact-card${confirming ? ' contact-card--confirming' : ''}`}>
      <div className="contact-card__avatar-wrap">
        <img
          src={avatar_url || '/img/default-avatar.svg'}
          alt=""
          className="contact-card__avatar"
        />
      </div>

      <div className="contact-card__info">
        {confirming ? (
          <span className="contact-card__confirm-text">Удалить {name}?</span>
        ) : (
          <>
            <Link to={`/profile/${id}`} className="contact-card__name">
              {name}
            </Link>
            <span className="contact-card__username">@{username}</span>
          </>
        )}
      </div>

      {confirming ? (
        <div className="contact-card__confirm-actions">
          <button
            className="contact-card__confirm-btn contact-card__confirm-btn--delete"
            onClick={handleConfirmRemove}
            disabled={removing}
          >
            {removing ? (
              <span className="contacts-modal__spinner" />
            ) : (
              'Удалить'
            )}
          </button>
          <button
            className="contact-card__confirm-btn contact-card__confirm-btn--cancel"
            onClick={(e) => {
              e.stopPropagation()
              cancelConfirm()
            }}
          >
            Отмена
          </button>
        </div>
      ) : (
        <div className="contact-card__actions">
          <button
            className="contact-card__btn contact-card__btn--chat"
            onClick={handleStartChat}
            disabled={starting}
            title="Написать сообщение"
          >
            {starting ? (
              <span className="contacts-modal__spinner" />
            ) : (
              <ChatBubbleIcon />
            )}
          </button>
          <button
            className="contact-card__btn contact-card__btn--remove"
            onClick={(e) => {
              e.stopPropagation()
              startConfirm()
            }}
            title="Удалить из контактов"
          >
            <CloseIcon />
          </button>
        </div>
      )}
    </li>
  )
}
