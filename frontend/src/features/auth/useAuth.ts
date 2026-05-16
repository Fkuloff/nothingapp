import { useCallback, useEffect, useRef, useState } from 'react'

import { endpoints } from '../../shared/api/endpoints'
import { getAuthToken, httpGet, setAuthToken } from '../../shared/api/httpClient'
import type { UserProfile } from '../../shared/api/types'
import { clearStoredAccountKey } from '../../shared/crypto/e2e'
import { clearPeerKeyCache } from '../../shared/crypto/peerKeys'
import { unregisterFCMDevice } from '../../shared/hooks/useFCMNotifications'

const CACHED_PROFILE_KEY = 'cached_user_profile'

function readCachedProfile(): UserProfile | null {
  try {
    const raw = localStorage.getItem(CACHED_PROFILE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as UserProfile
  } catch {
    return null
  }
}

function writeCachedProfile(profile: UserProfile | null) {
  try {
    if (profile === null) {
      localStorage.removeItem(CACHED_PROFILE_KEY)
    } else {
      localStorage.setItem(CACHED_PROFILE_KEY, JSON.stringify(profile))
    }
  } catch {
    // localStorage may be full or unavailable; non-fatal
  }
}

export function useAuth() {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const initializedRef = useRef(false)

  const refreshProfile = useCallback(async () => {
    try {
      // Only show loading spinner on initial load, not on refresh.
      // Setting loading=true unmounts ProtectedRoute's Outlet, closing any open modals.
      if (!initializedRef.current) setLoading(true)
      setError(null)
      const profile = await httpGet<UserProfile>(endpoints.auth.me)
      setUser(profile)
      writeCachedProfile(profile)
      initializedRef.current = true
    } catch (err) {
      console.error('Не удалось получить профиль', err)
      // Don't blow away cached profile on transient errors — only on auth failure,
      // which the caller signals by clearing the token. The cached profile lets the
      // shell render while we retry in the background.
      const cached = readCachedProfile()
      if (!cached) {
        setUser(null)
      }
      setError(err instanceof Error ? err.message : 'Не удалось получить профиль')
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      setLoading(true)
      await unregisterFCMDevice()
      await httpGet(endpoints.auth.logout)
    } catch (err) {
      console.error('Не удалось выйти из аккаунта', err)
      setError(err instanceof Error ? err.message : 'Не удалось выйти из аккаунта')
    } finally {
      setAuthToken(undefined)
      writeCachedProfile(null)
      // Best-effort scrub of other per-user cached state. Keys must match those used in
      // the consumers (ChatsPage uses `cached_chats_list`). Done here so a new login on
      // the same device doesn't briefly flash the previous user's data.
      try {
        localStorage.removeItem('cached_chats_list')
      } catch {
        // ignore
      }
      // Drop the E2E account_key from disk too — otherwise a new user signing in on
      // the same device could end up using the previous user's key to encrypt their
      // first messages (race between login and AccountKeyProvider's rehydrate).
      await clearStoredAccountKey()
      // Drop the in-memory peer public_key cache so a new user doesn't accidentally
      // see "trusted" entries left over from the previous session.
      clearPeerKeyCache()
      setUser(null)
      setLoading(false)
      initializedRef.current = false
    }
  }, [])

  // On mount: hydrate from the cached profile so the UI renders instantly, then refresh
  // from the API in the background. This avoids the "Loading profile…" splash on every
  // app launch — the user sees their last-known profile immediately and any changes
  // appear silently once the request resolves.
  useEffect(() => {
    const token = getAuthToken()
    if (!token) {
      setLoading(false)
      return
    }

    const cached = readCachedProfile()
    if (cached) {
      setUser(cached)
      setLoading(false)
      initializedRef.current = true
    }

    refreshProfile()
  }, [refreshProfile])

  return { user, loading, error, refreshProfile, logout }
}
