import { lazy, Suspense, type ComponentType } from 'react'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'
import AppLayout from '../App'
import { ProtectedRoute } from '../features/auth/ProtectedRoute'

const LoginPage = lazy(() => import('../pages/LoginPage'))
const RegisterPage = lazy(() => import('../pages/RegisterPage'))
const ChatsPage = lazy(() => import('../pages/ChatsPage'))
const ContactsPage = lazy(() => import('../pages/ContactsPage'))
const ProfilePage = lazy(() => import('../pages/ProfilePage'))
const SettingsPage = lazy(() => import('../pages/SettingsPage'))

const withSuspense = (Component: ComponentType) => (
  <Suspense fallback={<div>Loading...</div>}>
    <Component />
  </Suspense>
)

const router = createBrowserRouter([
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            path: '/',
            element: withSuspense(ChatsPage),
          },
          {
            path: '/contacts',
            element: withSuspense(ContactsPage),
          },
          {
            path: '/profile/:userId?',
            element: withSuspense(ProfilePage),
          },
          {
            path: '/settings',
            element: withSuspense(SettingsPage),
          },
        ],
      },
    ],
  },
  {
    path: '/login',
    element: withSuspense(LoginPage),
  },
  {
    path: '/register',
    element: withSuspense(RegisterPage),
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
