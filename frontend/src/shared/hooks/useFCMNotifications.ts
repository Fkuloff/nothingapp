import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { registerFCMToken, unregisterFCMToken } from '../api/pushApi'
import { getPlatform, isNative } from '../platform'

const STORED_TOKEN_KEY = 'fcm_token'

/**
 * Registers the device with FCM on native platforms, posts the token to the
 * backend, and navigates to the relevant chat when a notification is tapped.
 *
 * Noop on web — Web Push via VAPID handles that case.
 */
export function useFCMNotifications(enabled: boolean) {
  const navigate = useNavigate()

  useEffect(() => {
    if (!enabled || !isNative()) return

    let mounted = true
    let registrationListener: { remove: () => void } | undefined
    let errorListener: { remove: () => void } | undefined
    let actionListener: { remove: () => void } | undefined

    async function setup() {
      const { PushNotifications } = await import('@capacitor/push-notifications')

      const permStatus = await PushNotifications.checkPermissions()
      let receive = permStatus.receive
      if (receive === 'prompt' || receive === 'prompt-with-rationale') {
        const req = await PushNotifications.requestPermissions()
        receive = req.receive
      }
      if (receive !== 'granted') {
        console.warn('FCM: notification permission not granted')
        return
      }

      if (!mounted) return

      registrationListener = await PushNotifications.addListener('registration', async (token) => {
        try {
          const previous = localStorage.getItem(STORED_TOKEN_KEY)
          if (previous === token.value) return
          await registerFCMToken(token.value, getPlatform() as 'android' | 'ios')
          localStorage.setItem(STORED_TOKEN_KEY, token.value)
        } catch (err) {
          console.error('FCM: failed to register token with backend', err)
        }
      })

      errorListener = await PushNotifications.addListener('registrationError', (err) => {
        console.error('FCM registration error', err)
      })

      actionListener = await PushNotifications.addListener('pushNotificationActionPerformed', (action) => {
        const chatIdRaw = action.notification.data?.chat_id
        if (chatIdRaw) {
          navigate(`/?chat=${chatIdRaw}`)
        }
      })

      await PushNotifications.register()
    }

    void setup()

    return () => {
      mounted = false
      registrationListener?.remove()
      errorListener?.remove()
      actionListener?.remove()
    }
  }, [enabled, navigate])
}

/** Unregister the current device token (called on logout). */
export async function unregisterFCMDevice() {
  if (!isNative()) return
  const token = localStorage.getItem(STORED_TOKEN_KEY)
  if (!token) return
  try {
    await unregisterFCMToken(token)
  } catch (err) {
    console.warn('FCM: backend unregister failed', err)
  }
  localStorage.removeItem(STORED_TOKEN_KEY)
}
