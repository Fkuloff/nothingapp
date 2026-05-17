import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import './App.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AccountKeyProvider } from './features/auth/AccountKeyProvider'
import { AuthProvider } from './features/auth/AuthContext'
import { CallProvider } from './features/calls/CallProvider'
import { UpdateMandatoryScreen } from './features/updates/UpdateMandatoryScreen'
import { UpdateProvider } from './features/updates/UpdateProvider'
import { AppRouter } from './router/AppRouter'
import { hydrateAuthToken } from './shared/api/httpClient'
import { ToastProvider } from './shared/components/Toast'
import { ThemeProvider } from './shared/context/ThemeContext'
import { initEarlyPushHandlers } from './shared/earlyPush'
import { AndroidBackProvider } from './shared/hooks/useAndroidBack'

// Must run synchronously at module load, before hydrateAuthToken awaits, so we register
// the push action listener in time for Capacitor's cold-start buffered event.
initEarlyPushHandlers()

hydrateAuthToken().finally(() => {
  createRoot(document.getElementById('root') as HTMLElement).render(
    <StrictMode>
      <AndroidBackProvider>
        <ThemeProvider>
          <ToastProvider>
            <AccountKeyProvider>
              <AuthProvider>
                <CallProvider>
                  {/*
                    UpdateProvider lives ABOVE AuthProvider's gating so the
                    mandatory-update screen blocks even the login page — a
                    user on an unsupported build shouldn't be able to log
                    in (they may not be able to use the result anyway). The
                    Screen renders its children unchanged when no mandatory
                    update is in flight.
                  */}
                  <UpdateProvider>
                    <UpdateMandatoryScreen>
                      <AppRouter />
                    </UpdateMandatoryScreen>
                  </UpdateProvider>
                </CallProvider>
              </AuthProvider>
            </AccountKeyProvider>
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
