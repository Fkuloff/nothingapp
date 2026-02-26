import { PhoneIcon, PhoneOffIcon } from '../../shared/components/Icons'
import { useModalBehavior } from '../../shared/hooks/useModalBehavior'

type Props = {
  callerName: string
  callerAvatar?: string | null
  onAccept: () => void
  onReject: () => void
}

export function IncomingCallModal({ callerName, callerAvatar, onAccept, onReject }: Props) {
  const { handleBackdropClick } = useModalBehavior({ isOpen: true, onClose: onReject })

  return (
    <div className="incoming-call-backdrop" onClick={handleBackdropClick}>
      <div className="incoming-call-modal" role="dialog" aria-modal="true">
        <div className="incoming-call-modal__pulse" />
        <div className="incoming-call-modal__avatar">
          <img src={callerAvatar || '/img/default-avatar.svg'} alt={callerName} />
        </div>
        <div className="incoming-call-modal__name">{callerName}</div>
        <div className="incoming-call-modal__label">Входящий звонок...</div>
        <div className="incoming-call-modal__actions">
          <button className="incoming-call-modal__btn incoming-call-modal__btn--reject" onClick={onReject} aria-label="Отклонить">
            <PhoneOffIcon size={24} />
          </button>
          <button className="incoming-call-modal__btn incoming-call-modal__btn--accept" onClick={onAccept} aria-label="Принять">
            <PhoneIcon size={24} />
          </button>
        </div>
      </div>
    </div>
  )
}
