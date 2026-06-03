// TS wrapper around the native ShareTargetPlugin (Kotlin). Unregistered on
// web/iOS, so guard callers with isNative().

import type { PluginListenerHandle } from '@capacitor/core'
import { registerPlugin } from '@capacitor/core'

export interface SharedItem {
  text: string | null
}

export interface ShareTargetPlugin {
  /** Drains the cold-start share intent (one-shot); {text: null} if none. */
  getSharedItem(): Promise<SharedItem>
  /** Fires when a share arrives while the app is already running. */
  addListener(
    eventName: 'shareReceived',
    listener: (data: { text: string }) => void,
  ): Promise<PluginListenerHandle>
}

export const ShareTarget = registerPlugin<ShareTargetPlugin>('ShareTarget')
