import { useCallback, useEffect, useState } from 'react'
import { httpGet, setAuthToken } from '../../shared/api/httpClient'
import type { UserProfile } from '../../shared/api/types'
import { endpoints } from '../../shared/api/endpoints'

export function useAuth() {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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
    }
  }, [])

  useEffect(() => {
    refreshProfile()
  }, [refreshProfile])

  return { user, loading, error, refreshProfile, logout }
}
