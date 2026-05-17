import { createContext, useContext } from 'react'

import type { AppRelease } from '../../shared/api/updatesApi'

/**
 * State machine for the in-app updater. Transitions:
 *
 *   loading ─▶ up_to_date          (cold-start check found nothing newer)
 *           ─▶ available           (newer version found, soft)
 *           ─▶ mandatory           (newer version found AND current < min_supported)
 *           ─▶ error               (transport failure — silent for soft check)
 *
 *   available ─▶ downloading ─▶ ready_to_install
 *                            ─▶ error            (sha mismatch / network)
 *   mandatory ─▶ (same as available)
 *
 *   ready_to_install ─▶ (Android takes over; our process is killed)
 *
 * `dismiss()` from `available` goes back to `up_to_date` for THIS version
 * code only — Preferences key `update_dismissed_version_code` remembers it
 * so the banner doesn't reappear until a newer release ships.
 */
export type UpdateState =
  | { status: 'loading' }
  | { status: 'up_to_date'; currentVersionCode: number }
  | { status: 'available'; release: AppRelease; currentVersionCode: number; mandatory: false }
  | { status: 'available'; release: AppRelease; currentVersionCode: number; mandatory: true }
  | {
      status: 'downloading'
      release: AppRelease
      currentVersionCode: number
      mandatory: boolean
      progress: { loaded: number; total: number }
    }
  | {
      status: 'ready_to_install'
      release: AppRelease
      path: string
      currentVersionCode: number
      mandatory: boolean
    }
  | { status: 'error'; message: string; currentVersionCode: number | null; mandatory: boolean }

export type UpdateContextValue = {
  state: UpdateState
  /** Force-fetch /api/updates/latest, ignoring the 24h debounce. */
  checkNow: () => Promise<void>
  /**
   * Mark the currently-offered version as dismissed; the banner won't show
   * again until a newer release ships. No-op for mandatory updates.
   */
  dismiss: () => Promise<void>
  /** Kick off the download → install flow. No-op outside 'available'/'mandatory'. */
  startDownload: () => Promise<void>
  /** Hand the downloaded APK to the system PackageInstaller. */
  install: () => Promise<void>
}

export const UpdateContext = createContext<UpdateContextValue | null>(null)

export function useUpdate(): UpdateContextValue {
  const ctx = useContext(UpdateContext)
  if (!ctx) throw new Error('useUpdate must be used inside UpdateProvider')
  return ctx
}
