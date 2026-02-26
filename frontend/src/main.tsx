import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import './App.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AuthProvider } from './features/auth/AuthContext'
import { CallProvider } from './features/calls/CallProvider'
import { AppRouter } from './router/AppRouter'
import { ToastProvider } from './shared/components/Toast'
import { ThemeProvider } from './shared/context/ThemeContext'

createRoot(document.getElementById('root') as HTMLElement).render(
  <StrictMode>
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <CallProvider>
            <AppRouter />
          </CallProvider>
        </AuthProvider>
      </ToastProvider>
    </ThemeProvider>
  </StrictMode>,
)

// Register service worker for push notifications
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch((err) => {
    console.error('Service worker registration failed:', err)
  })
}
