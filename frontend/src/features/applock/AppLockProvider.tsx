import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import {
  getAppLockTimeoutMs,
  isAppLockEnabled,
  recordLastActive,
  setAppLockEnabled,
  setAppLockTimeoutMs,
  shouldLockNow,
} from '../../shared/appLock'
import { biometricAuthenticate, biometricAvailable } from '../../shared/nativeBiometric'
import { getPlatform } from '../../shared/platform'
import { AppLockContext, type AppLockContextValue } from './AppLockContext'
import { EnableAppLockOffer } from './EnableAppLockOffer'
import { LockScreen } from './LockScreen'

/**
 * Hosts the app-lock state machine: lock on cold start / return from
 * background after the autolock timeout, unlock via LockScreen (biometric or
 * password). Children stay mounted under the lock overlay so the WebSocket
 * keeps running — the lock hides the UI, it doesn't suspend the app.
 *
 * Android-only: on web/desktop the provider is inert (never locks) and the
 * settings UI for it isn't rendered.
 */
export function AppLockProvider({ children }: { children: ReactNode }) {
  const isAndroid = getPlatform() === 'android'
  // localStorage is hydrated from Preferences before the first render
  // (main.tsx), so this synchronous init is authoritative — locked content
  // never flashes before the gate appears.
  const [locked, setLocked] = useState(() => isAndroid && shouldLockNow())
  const [enabled, setEnabled] = useState(() => isAppLockEnabled())
  const [timeoutMs, setTimeoutMsState] = useState(() => getAppLockTimeoutMs())
  const [biometricsOk, setBiometricsOk] = useState(false)
  // The listener below decides with current values; refs dodge re-subscribing
  // the native listener on every settings change.
  const lockedRef = useRef(locked)
  lockedRef.current = locked

  useEffect(() => {
    if (!isAndroid) return
    void biometricAvailable().then(setBiometricsOk)
  }, [isAndroid])

  useEffect(() => {
    if (!isAndroid) return
    let remove: (() => void) | undefined
    void (async () => {
      const { App } = await import('@capacitor/app')
      const handle = await App.addListener('appStateChange', ({ isActive }) => {
        if (!isActive) {
          // Don't refresh the activity stamp while the lock screen is up —
          // otherwise "lock → background → quick return" would unlock a
          // session the user never re-verified.
          if (!lockedRef.current) recordLastActive()
          return
        }
        if (shouldLockNow()) setLocked(true)
      })
      remove = () => void handle.remove()
    })()
    return () => remove?.()
  }, [isAndroid])

  const unlock = useCallback(() => {
    recordLastActive()
    setLocked(false)
  }, [])

  const enable = useCallback(async () => {
    const verified = await biometricAuthenticate(
      'Вход по биометрии',
      'Подтвердите, чтобы включить блокировку приложения',
    )
    if (!verified) return false
    setAppLockEnabled(true)
    recordLastActive()
    setEnabled(true)
    setTimeoutMsState(getAppLockTimeoutMs())
    return true
  }, [])

  const disable = useCallback(() => {
    setAppLockEnabled(false)
    setEnabled(false)
    setLocked(false)
  }, [])

  const setTimeoutMs = useCallback((ms: number) => {
    setAppLockTimeoutMs(ms)
    setTimeoutMsState(ms)
  }, [])

  const value = useMemo<AppLockContextValue>(
    () => ({ locked, enabled, biometricsAvailable: biometricsOk, timeoutMs, unlock, enable, disable, setTimeoutMs }),
    [locked, enabled, biometricsOk, timeoutMs, unlock, enable, disable, setTimeoutMs],
  )

  return (
    <AppLockContext.Provider value={value}>
      {children}
      {locked && <LockScreen />}
      {isAndroid && !locked && <EnableAppLockOffer />}
    </AppLockContext.Provider>
  )
}
