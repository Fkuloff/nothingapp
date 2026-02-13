import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css'
import './App.css'
import { AppRouter } from './router/AppRouter'
import { AuthProvider } from './features/auth/AuthContext'
import { ToastProvider } from './shared/components/Toast'
import { ThemeProvider } from './shared/context/ThemeContext'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <AppRouter />
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
