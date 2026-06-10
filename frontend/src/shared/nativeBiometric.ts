// Thin TS wrapper around the custom BiometricPlugin (Kotlin) living in
// `frontend/android/app/.../BiometricPlugin.kt`.
//
// Android-only: on web/desktop the plugin isn't registered and both helpers
// degrade to "not available" / "not verified", so callers don't need their
// own platform guards beyond what reads naturally.

import { registerPlugin } from '@capacitor/core'

import { getPlatform } from './platform'

interface BiometricPlugin {
  /** Reports whether the device has enrolled class-2+ biometrics. */
  isAvailable(): Promise<{ available: boolean }>
  /**
   * Shows the system BiometricPrompt. Resolves on success; rejects on user
   * cancel, lockout or hardware error. Single failed attempts do NOT reject —
   * the system prompt keeps itself open for retries.
   */
  authenticate(options?: { title?: string; subtitle?: string; cancelTitle?: string }): Promise<void>
}

const Biometric = registerPlugin<BiometricPlugin>('Biometric')

export async function biometricAvailable(): Promise<boolean> {
  if (getPlatform() !== 'android') return false
  try {
    return (await Biometric.isAvailable()).available
  } catch {
    return false
  }
}

/** True — личность подтверждена; false — отмена/локаут/ошибка. */
export async function biometricAuthenticate(title: string, subtitle?: string): Promise<boolean> {
  if (getPlatform() !== 'android') return false
  try {
    await Biometric.authenticate({ title, subtitle, cancelTitle: 'Отмена' })
    return true
  } catch {
    return false
  }
}
