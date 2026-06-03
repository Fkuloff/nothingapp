import { useEffect } from 'react'

import { setPendingShare } from '../pendingShare'
import { isNative } from '../platform'
import { ShareTarget } from '../shareTarget'

/**
 * Drains the cold-start share intent and subscribes to warm-start shares,
 * funneling both into setPendingShare. No-op on web/iOS. Call once after login.
 */
export function useShareTarget() {
  useEffect(() => {
    if (!isNative()) return
    let cancelled = false
    let handle: { remove: () => void } | undefined

    // Cold start: the share intent launched the app.
    ShareTarget.getSharedItem()
      .then(({ text }) => { if (!cancelled && text) setPendingShare(text) })
      .catch(() => { /* plugin missing / no share — ignore */ })

    // Warm start: a share arrived while the app was already running.
    ShareTarget.addListener('shareReceived', ({ text }) => {
      if (text) setPendingShare(text)
    }).then((h) => {
      if (cancelled) h.remove()
      else handle = h
    }).catch(() => { /* ignore */ })

    return () => {
      cancelled = true
      handle?.remove()
    }
  }, [])
}
