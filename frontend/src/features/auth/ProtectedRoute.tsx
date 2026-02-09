import { Navigate, Outlet } from 'react-router-dom'
import { useAuthContext } from './AuthContext'

export function ProtectedRoute() {
  const { user, loading } = useAuthContext()

  if (loading) {
    return <div className="d-flex justify-content-center p-5 text-muted">Загрузка профиля...</div>
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
