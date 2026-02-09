import { Outlet, Link, useNavigate } from 'react-router-dom'
import './App.css'
import { useAuthContext } from './features/auth/AuthContext'

export default function AppLayout() {
  const { user, logout } = useAuthContext()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="app-shell">
      <header className="topbar navbar navbar-expand-md navbar-dark">
        <div className="container-fluid">
          <Link to="/" className="navbar-brand fw-bold topbar__brand">
            <span className="logo-dot" />
            <span>Pulse Messenger</span>
          </Link>

          <div className="d-flex align-items-center ms-auto gap-3 topbar__actions">
            {user && (
              <>
                <Link to="/profile" className="nav-link topbar__link">
                  <span className="avatar avatar-sm me-2">
                    <img src={user.avatar_url ?? '/static/img/default-avatar.svg'} alt="avatar" />
                  </span>
                  <span>{user.name || user.username}</span>
                </Link>
                <button className="btn btn-outline-light btn-sm" onClick={handleLogout}>
                  Log out
                </button>
              </>
            )}
          </div>
        </div>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
    </div>
  )
}


