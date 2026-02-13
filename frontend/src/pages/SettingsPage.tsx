import { useOutletContext } from 'react-router-dom'
import { useTheme } from '../shared/hooks/useTheme'
import { usePushNotifications } from '../shared/hooks/usePushNotifications'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import type { OutletContextType } from '../App'

export default function SettingsPage() {
  const { setMenuOpen } = useOutletContext<OutletContextType>()
  const { theme, setTheme } = useTheme()
  const {
    isSupported: pushSupported,
    isSubscribed: pushSubscribed,
    isLoading: pushLoading,
    permission: pushPermission,
    subscribe: pushSubscribe,
    unsubscribe: pushUnsubscribe,
  } = usePushNotifications()

  const handlePushToggle = async () => {
    if (pushSubscribed) {
      await pushUnsubscribe()
    } else {
      await pushSubscribe()
    }
  }

  return (
    <div className="page-container">
      {/* Header */}
      <div className="page-header">
        <HamburgerButton onClick={() => setMenuOpen(true)} />
        <h2>Настройки</h2>
      </div>

      {/* Content */}
      <div className="page-content">
        {/* Appearance section */}
        <div className="settings-section">
          <h3>Внешний вид</h3>
          <div className="settings-item">
            <span>Тема</span>
            <select
              value={theme}
              onChange={(e) => setTheme(e.target.value as 'light' | 'dark')}
            >
              <option value="dark">Тёмная</option>
              <option value="light">Светлая</option>
            </select>
          </div>
        </div>

        {/* Notifications section */}
        <div className="settings-section">
          <h3>Уведомления</h3>
          <div className="settings-item" style={{ opacity: 0.5 }}>
            <span>Звуковые уведомления</span>
            <span className="chip">Скоро</span>
          </div>
          <div className="settings-item">
            <span>Push-уведомления</span>
            {!pushSupported ? (
              <span className="chip">Не поддерживается</span>
            ) : pushPermission === 'denied' ? (
              <span className="chip">Заблокировано</span>
            ) : (
              <button
                className={`btn btn-sm ${pushSubscribed ? 'btn-success' : 'btn-outline-secondary'}`}
                onClick={handlePushToggle}
                disabled={pushLoading}
              >
                {pushLoading ? '...' : pushSubscribed ? 'Вкл' : 'Выкл'}
              </button>
            )}
          </div>
        </div>

        {/* Privacy section - placeholder */}
        <div className="settings-section">
          <h3>Приватность</h3>
          <div className="settings-item" style={{ opacity: 0.5 }}>
            <span>Показывать статус онлайн</span>
            <span className="chip">Скоро</span>
          </div>
          <div className="settings-item" style={{ opacity: 0.5 }}>
            <span>Показывать время последнего визита</span>
            <span className="chip">Скоро</span>
          </div>
        </div>

        {/* About section */}
        <div className="settings-section">
          <h3>О приложении</h3>
          <div className="settings-item">
            <span>Версия</span>
            <span className="text-muted">1.0.0</span>
          </div>
        </div>
      </div>
    </div>
  )
}
