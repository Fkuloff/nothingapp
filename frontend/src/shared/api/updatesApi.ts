import { endpoints } from './endpoints'
import { httpGet } from './httpClient'

/**
 * Wire shape of one row in `app_releases` as the backend returns it from
 * GET /api/updates/latest. Mirrors `backend/internal/models/app_release.go`.
 *
 * Fields stripped from the model (CreatedAt / UpdatedAt / DeletedAt) are
 * GORM internals and not part of the public API surface.
 */
export type AppRelease = {
  ID: number
  platform: string
  version_name: string
  version_code: number
  min_supported_version_code: number
  url: string
  sha256: string
  size_bytes: number
  changelog: string
  released_at: string
}

/**
 * Fetches the latest registered release for the platform. Returns null when
 * the server replies 204 No Content (no releases registered yet) — caller
 * should treat that as "no update available".
 *
 * Throws on transport errors / 5xx — caller is expected to silently swallow
 * those (we don't want a transient backend hiccup to surface as a UI error).
 */
export async function fetchLatestRelease(platform = 'android'): Promise<AppRelease | null> {
  const result = await httpGet<AppRelease | undefined>(
    `${endpoints.updates.latest}?platform=${encodeURIComponent(platform)}`,
  )
  return result ?? null
}
