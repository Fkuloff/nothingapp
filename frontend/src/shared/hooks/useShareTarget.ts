import { useEffect } from 'react'

import { setPendingShare } from '../pendingShare'
import { isNative } from '../platform'
import { ShareTarget } from '../shareTarget'

/**
 * Wires the native Android Share Target into the app. On mount:
 *   - drains any cold-start share intent (getSharedItem),
 *   - subscribes to warm-start "shareReceived" events.
 * Both funnel into setPendingShare, which ChatsPage turns into a chat-picker.
 *
 * No-op on web / iOS — the ShareTarget plugin isn't registered there, so we
 * never touch it. Mirrors the shape of useFCMNotifications.
 */
export function useShareTarget() {
  useEffect(() => {
    if (!isNative()) return
    let cancelled = false
    let handle: { remove: () => void } | undefined

    // Cold start: the share intent launched the app. Drain it once the WebView
    // (and this hook) is up.
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
