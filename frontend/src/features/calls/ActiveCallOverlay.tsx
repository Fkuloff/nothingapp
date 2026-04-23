import { resolveApiUrl } from '../../shared/api/httpClient'
import { CloseIcon, MicIcon, MicOffIcon } from '../../shared/components/Icons'

type Props = {
  otherUsername: string
  otherAvatar?: string | null
  duration: number
  isMuted: boolean
  status: 'outgoing' | 'active'
  onToggleMute: () => void
  onHangup: () => void
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, '0')
  const s = (seconds % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

export function ActiveCallOverlay({ otherUsername, otherAvatar, duration, isMuted, status, onToggleMute, onHangup }: Props) {
  return (
    <div className="active-call-overlay">
      <div className="active-call-overlay__info">
        <img
          className="active-call-overlay__avatar"
          src={resolveApiUrl(otherAvatar) || '/img/default-avatar.svg'}
          alt={otherUsername}
        />
        <div className="active-call-overlay__text">
          <span className="active-call-overlay__name">{otherUsername}</span>
          <span className="active-call-overlay__status">
            {status === 'outgoing' ? 'Вызов...' : formatDuration(duration)}
          </span>
        </div>
      </div>
      <div className="active-call-overlay__controls">
        <button
          className={`active-call-overlay__btn${isMuted ? ' active-call-overlay__btn--active' : ''}`}
          onClick={onToggleMute}
          aria-label={isMuted ? 'Включить микрофон' : 'Выключить микрофон'}
        >
          {isMuted ? <MicOffIcon size={18} /> : <MicIcon size={18} />}
        </button>
        <button
          className="active-call-overlay__btn active-call-overlay__btn--hangup"
          onClick={onHangup}
          aria-label="Завершить звонок"
        >
          <CloseIcon size={18} />
        </button>
      </div>
    </div>
  )
}
