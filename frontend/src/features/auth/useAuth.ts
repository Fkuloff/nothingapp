import { useCallback, useEffect, useState } from 'react'
import { httpGet, setAuthToken } from '../../shared/api/httpClient'
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

  const refreshProfile = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const profile = await httpGet<UserProfile>(endpoints.auth.me)
      setUser(profile)
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
      // Note: We don't clear crypto data on logout to preserve keys for re-login.
      // Keys are cleared only when explicitly requested via key management UI.
    }
  }, [])

  // Initialize E2E encryption keys when user is authenticated
  useEffect(() => {
    if (!user) {
      setCryptoReady(false)
      setNeedsKeyRestore(false)
      return
    }

    initializeKeys()
      .then(({ ready, needsRestore }) => {
        setCryptoReady(ready)
        setNeedsKeyRestore(needsRestore)
      })
      .catch((err) => {
        console.error('Failed to initialize E2E keys:', err)
        setCryptoReady(false)
      })
  }, [user])

  useEffect(() => {
    refreshProfile()
  }, [refreshProfile])

  return { user, loading, error, cryptoReady, needsKeyRestore, refreshProfile, logout }
}
