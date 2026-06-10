import { createContext, useContext } from 'react'

export type AppLockContextValue = {
  /** Lock screen is currently shown over the app. */
  locked: boolean
  /** App-lock (biometric + autolock) is switched on for this device. */
  enabled: boolean
  /** Device has enrolled biometrics (false until the async probe answers). */
  biometricsAvailable: boolean
  timeoutMs: number
  /** Dismiss the lock screen after a successful biometric/password check. */
  unlock: () => void
  /**
   * Runs the system biometric prompt as a test and, on success, turns the
   * lock on. Returns false when the user cancels or the prompt errors.
   */
  enable: () => Promise<boolean>
  disable: () => void
  setTimeoutMs: (ms: number) => void
}

export const AppLockContext = createContext<AppLockContextValue | null>(null)

export function useAppLock(): AppLockContextValue {
  const ctx = useContext(AppLockContext)
  if (!ctx) throw new Error('useAppLock must be used inside AppLockProvider')
  return ctx
}
