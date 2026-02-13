import { usePushNotifications } from '../hooks/usePushNotifications'

type PushToggleProps = {
  /** CSS class for "not supported" / "blocked" labels */
  placeholderClass?: string
}

export function PushToggle({ placeholderClass }: PushToggleProps) {
  const {
    isSupported,
    isSubscribed,
    isLoading,
    permission,
    subscribe,
    unsubscribe,
  } = usePushNotifications()

  const handleToggle = async () => {
    if (isSubscribed) {
      await unsubscribe()
    } else {
      await subscribe()
    }
  }

  if (!isSupported) {
    return <span className={placeholderClass}>Не поддерживается</span>
  }

  if (permission === 'denied') {
    return <span className={placeholderClass}>Заблокировано в браузере</span>
  }

  return (
    <button
      className={`btn btn-sm ${isSubscribed ? 'btn-success' : 'btn-outline-secondary'}`}
      onClick={handleToggle}
      disabled={isLoading}
    >
      {isLoading ? '...' : isSubscribed ? 'Вкл' : 'Выкл'}
    </button>
  )
}
