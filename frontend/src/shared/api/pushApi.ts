import { endpoints } from './endpoints'
import { httpGet, httpPost } from './httpClient'

type VAPIDKeyResponse = {
  vapid_public_key: string
}

type PushStatusResponse = {
  enabled: boolean
  has_subscription: boolean
}

/** Convert ArrayBuffer to base64url (RFC 4648 §5), required by webpush-go */
function arrayBufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export function getVAPIDKey() {
  return httpGet<VAPIDKeyResponse>(endpoints.push.vapidKey)
}

export function subscribePush(subscription: PushSubscription) {
  const key = subscription.getKey('p256dh')
  const auth = subscription.getKey('auth')

  if (!key || !auth) {
    throw new Error('Invalid push subscription keys')
  }

  return httpPost(endpoints.push.subscribe, {
    endpoint: subscription.endpoint,
    p256dh: arrayBufferToBase64url(key),
    auth: arrayBufferToBase64url(auth),
  })
}

export function unsubscribePush(endpoint: string) {
  return httpPost(endpoints.push.unsubscribe, { endpoint })
}

export function getPushStatus() {
  return httpGet<PushStatusResponse>(endpoints.push.status)
}

export function registerFCMToken(token: string, platform: 'android' | 'ios') {
  return httpPost(endpoints.push.fcmRegister, { token, platform })
}

export function unregisterFCMToken(token: string) {
  return httpPost(endpoints.push.fcmUnregister, { token })
}
