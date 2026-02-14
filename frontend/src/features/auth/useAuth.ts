import { useCallback, useEffect, useRef, useState } from 'react'
import { getAuthToken, httpGet, setAuthToken } from '../../shared/api/httpClient'
import type { UserProfile } from '../../shared/api/types'
import { endpoints } from '../../shared/api/endpoints'
import { initializeKeys } from '../../shared/crypto/keyExchange'
import { clearAllCryptoData } from '../../shared/crypto/keyStore'

export function useAuth() {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cryptoReady, setCryptoReady] = useState(false)
  const [needsKeyRestore, setNeedsKeyRestore] = useState(false)
  const [needsBackupFirst, setNeedsBackupFirst] = useState(false)
  const initializedRef = useRef(false)

  const refreshProfile = useCallback(async () => {
    try {
      // Only show loading spinner on initial load, not on refresh.
      // Setting loading=true unmounts ProtectedRoute's Outlet, closing any open modals.
      if (!initializedRef.current) setLoading(true)
      setError(null)
      const profile = await httpGet<UserProfile>(endpoints.auth.me)
      setUser(profile)
      initializedRef.current = true
    } catch (err) {
      console.error('Не удалось получить профиль', err)
      setUser(null)
      setError(err instanceof Error ? err.message : 'Не удалось получить профиль')
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      setLoading(true)
      await httpGet(endpoints.auth.logout)
    } catch (err) {
      console.error('Не удалось выйти из аккаунта', err)
      setError(err instanceof Error ? err.message : 'Не удалось выйти из аккаунта')
    } finally {
      setAuthToken(undefined)
      setUser(null)
      setLoading(false)
      initializedRef.current = false
      // Clear crypto keys so they don't leak to the next user session
      clearAllCryptoData().catch(() => {})
      localStorage.removeItem('crypto_owner_id')
    }
  }, [])

  // On mount: check if we have a token and try to restore the session
  useEffect(() => {
    const token = getAuthToken()
    if (token) {
      refreshProfile()
    } else {
      setLoading(false)
    }
  }, [refreshProfile])

  // Initialize E2E encryption keys when user is authenticated
  useEffect(() => {
    if (!user) {
      setCryptoReady(false)
      setNeedsKeyRestore(false)
      setNeedsBackupFirst(false)
      return
    }

    initializeKeys(user.id)
      .then((result) => {
        setCryptoReady(result.ready)
        setNeedsKeyRestore(result.needsRestore)
        setNeedsBackupFirst(result.needsBackupFirst)
      })
      .catch((err) => {
        console.error('Failed to initialize E2E keys:', err)
        setCryptoReady(false)
      })
  }, [user])

  return { user, loading, error, cryptoReady, needsKeyRestore, needsBackupFirst, refreshProfile, logout }
}
