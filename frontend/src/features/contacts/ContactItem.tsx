import { Link } from 'react-router-dom'

type Props = {
  id: number
  username: string
  name: string
  avatar_url?: string
  onStartChat: (userId: number) => void
  onRemove: (userId: number) => void
}

export function ContactItem({ id, username, name, avatar_url, onStartChat, onRemove }: Props) {
  return (
    <li className="chat-list-item">
      <span className="avatar avatar-md">
        <img src={avatar_url || '/img/default-avatar.svg'} alt="Avatar" />
      </span>

      <div className="chat-list-item-content">
        <div className="chat-list-item__top">
          <Link to={`/profile/${id}`} className="chat-list-item__name">
            {name}
          </Link>
        </div>
        <div className="chat-list-item__preview">
          <span className="text-muted">@{username}</span>
        </div>
      </div>

      <div className="d-flex gap-2">
        <button
          className="btn btn-sm btn-outline-secondary"
          onClick={(e) => {
            e.stopPropagation()
            onStartChat(id)
          }}
        >
          Чат
        </button>
        <button
          className="btn btn-sm btn-outline-danger"
          onClick={(e) => {
            e.stopPropagation()
            onRemove(id)
          }}
        >
          Удалить
        </button>
      </div>
    </li>
  )
}
