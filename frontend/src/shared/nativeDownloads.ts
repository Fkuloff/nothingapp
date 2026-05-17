// Thin TS wrapper around the custom DownloadsPlugin (Java) that lives in
// `frontend/android/app/.../DownloadsPlugin.java`.
//
// The plugin writes a base64 blob to the system Downloads folder via
// MediaStore.Downloads — the only API that lets us land a file in
// /Download/ on Android 11+ without MANAGE_EXTERNAL_STORAGE.
//
// On Android < 10 the plugin rejects with `unsupported_api`; on iOS the
// plugin is simply unregistered and any call rejects. Either way the
// downloadAttachment.ts caller falls back to the existing Filesystem
// chain. No-op on web.

import { registerPlugin } from '@capacitor/core'

export interface DownloadsPlugin {
  /**
   * Save the given base64-encoded bytes into the system Downloads folder.
   * Resolves with the content:// uri and a human-readable "Download/<name>"
   * label for toast display. Rejects on:
   *   - unsupported_api  (Android < 10 — caller should fall back)
   *   - MediaStore.insert returned null
   *   - openOutputStream / write failed (disk full, IO error)
   */
  saveToDownloads(options: {
    data: string
    fileName: string
    mimeType?: string
  }): Promise<{ uri: string; path: string }>
}

export const Downloads = registerPlugin<DownloadsPlugin>('Downloads')
