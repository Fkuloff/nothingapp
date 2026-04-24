// Register the push-notification-tap listener at app bootstrap, before any other async work,
// so Capacitor's buffered event (fired when the app is cold-started by a notification tap)
// reaches us instead of being dropped.
//
// The @capacitor/push-notifications plugin is known to lose the tap event on cold start /
// long-background if the listener is registered lazily inside a React effect — by then the
// native side has already flushed and discarded the pending event.

import { setPendingChat } from './pendingChat'
import { isNative } from './platform'

let initialized = false

export function initEarlyPushHandlers() {
  if (initialized || !isNative()) return
  initialized = true

  void (async () => {
    try {
      const { PushNotifications } = await import('@capacitor/push-notifications')
      await PushNotifications.addListener('pushNotificationActionPerformed', (action) => {
        const chatIdRaw = action.notification.data?.chat_id
        if (!chatIdRaw) return
        const chatId = Number(chatIdRaw)
        if (!Number.isFinite(chatId)) return
        setPendingChat(chatId)
      })
    } catch (err) {
      console.error('earlyPush: failed to register action listener', err)
    }
  })()
}
