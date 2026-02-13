import { useState, useEffect, useCallback } from 'react'
import { getVAPIDKey, subscribePush, unsubscribePush, getPushStatus } from '../api/pushApi'

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = atob(base64)
  return Uint8Array.from(rawData, (char) => char.charCodeAt(0))
}

export function usePushNotifications() {
  const [isSupported, setIsSupported] = useState(false)
  const [isSubscribed, setIsSubscribed] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [permission, setPermission] = useState<NotificationPermission>('default')

  useEffect(() => {
    const supported = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
    setIsSupported(supported)

    if (!supported) {
      setIsLoading(false)
      return
    }

    setPermission(Notification.permission)

    getPushStatus()
      .then((status) => {
        setIsSubscribed(status.has_subscription)
      })
      .catch(() => {
        // Not authenticated or push not configured
      })
      .finally(() => setIsLoading(false))
  }, [])

  const subscribe = useCallback(async () => {
    if (!isSupported) return false

    try {
      setIsLoading(true)

      const perm = await Notification.requestPermission()
      setPermission(perm)
      if (perm !== 'granted') return false

      const registration = await navigator.serviceWorker.register('/sw.js')
      await navigator.serviceWorker.ready

      const { vapid_public_key } = await getVAPIDKey()

      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapid_public_key).buffer as ArrayBuffer,
      })

      await subscribePush(subscription)
      setIsSubscribed(true)
      return true
    } catch (error) {
      console.error('Failed to subscribe to push notifications:', error)
      return false
    } finally {
      setIsLoading(false)
    }
  }, [isSupported])

  const unsubscribe = useCallback(async () => {
    try {
      setIsLoading(true)

      const registration = await navigator.serviceWorker.getRegistration()
      if (registration) {
        const subscription = await registration.pushManager.getSubscription()
        if (subscription) {
          await unsubscribePush(subscription.endpoint)
          await subscription.unsubscribe()
        }
      }

      setIsSubscribed(false)
      return true
    } catch (error) {
      console.error('Failed to unsubscribe from push notifications:', error)
      return false
    } finally {
      setIsLoading(false)
    }
  }, [])

  return {
    isSupported,
    isSubscribed,
    isLoading,
    permission,
    subscribe,
    unsubscribe,
  }
}
