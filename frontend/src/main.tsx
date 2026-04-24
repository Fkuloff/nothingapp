import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import './App.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AuthProvider } from './features/auth/AuthContext'
import { CallProvider } from './features/calls/CallProvider'
import { AppRouter } from './router/AppRouter'
import { hydrateAuthToken } from './shared/api/httpClient'
import { ToastProvider } from './shared/components/Toast'
import { ThemeProvider } from './shared/context/ThemeContext'
import { AndroidBackProvider } from './shared/hooks/useAndroidBack'

hydrateAuthToken().finally(() => {
  createRoot(document.getElementById('root') as HTMLElement).render(
    <StrictMode>
      <AndroidBackProvider>
        <ThemeProvider>
          <ToastProvider>
            <AuthProvider>
              <CallProvider>
                <AppRouter />
              </CallProvider>
            </AuthProvider>
          </ToastProvider>
        </ThemeProvider>
      </AndroidBackProvider>
    </StrictMode>,
  )
})

// Register service worker for push notifications
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch((err) => {
    console.error('Service worker registration failed:', err)
  })
}
