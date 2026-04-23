import { type ComponentType,lazy, Suspense } from 'react'
import { createBrowserRouter, createHashRouter, Navigate, RouterProvider } from 'react-router-dom'

import AppLayout from '../App'
import { ProtectedRoute } from '../features/auth/ProtectedRoute'
import { isNative } from '../shared/platform'

const LoginPage = lazy(() => import('../pages/LoginPage'))
const RegisterPage = lazy(() => import('../pages/RegisterPage'))
const ChatsPage = lazy(() => import('../pages/ChatsPage'))
const ProfilePage = lazy(() => import('../pages/ProfilePage'))

const withSuspense = (Component: ComponentType) => (
  <Suspense fallback={<div>Loading...</div>}>
    <Component />
  </Suspense>
)

const createRouter = isNative() ? createHashRouter : createBrowserRouter

const router = createRouter([
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
            path: '/profile/:userId?',
            element: withSuspense(ProfilePage),
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
