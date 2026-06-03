// Thin TS wrapper around the custom ShareTargetPlugin (Kotlin) living in
// `frontend/android/app/.../ShareTargetPlugin.kt`.
//
// No-op on web / iOS (the plugin isn't registered there) — callers must guard
// with isNative() before reaching for any of this. Powers "Share to Messenger"
// from the Android system share-sheet (ACTION_SEND, text/plain).

import type { PluginListenerHandle } from '@capacitor/core'
import { registerPlugin } from '@capacitor/core'

export interface SharedItem {
  /** Plain text / URL from the share intent, or null when nothing is pending. */
  text: string | null
}

export interface ShareTargetPlugin {
  /**
   * Drains the cold-start share intent (one-shot). Resolves {text: null} when
   * the app wasn't launched from a share.
   */
  getSharedItem(): Promise<SharedItem>

  /**
   * Warm-start deliveries: fires when a share arrives while the app is already
   * running (routed through MainActivity.onNewIntent → plugin).
   */
  addListener(
    eventName: 'shareReceived',
    listener: (data: { text: string }) => void,
  ): Promise<PluginListenerHandle>
}

export const ShareTarget = registerPlugin<ShareTargetPlugin>('ShareTarget')
