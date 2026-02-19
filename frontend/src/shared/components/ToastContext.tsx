import { createContext, useContext } from 'react'

type ToastType = 'success' | 'error' | 'warning' | 'info'

type ToastContextValue = {
  showToast: (message: string, type?: ToastType) => void
}

export const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast(): ToastContextValue {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within ToastProvider')
  }
  return context
}
