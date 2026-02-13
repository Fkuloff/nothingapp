import { useOutletContext } from 'react-router-dom'
import { useTheme } from '../shared/hooks/useTheme'
import { PushToggle } from '../shared/components/PushToggle'
import { HamburgerButton } from '../features/menu/HamburgerButton'
import type { OutletContextType } from '../App'

export default function SettingsPage() {
  const { setMenuOpen } = useOutletContext<OutletContextType>()
  const { theme, setTheme } = useTheme()

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
            <PushToggle placeholderClass="chip" />
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
