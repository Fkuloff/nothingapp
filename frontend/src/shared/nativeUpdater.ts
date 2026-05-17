// Thin TS wrapper around the custom UpdaterPlugin (Kotlin) living in
// `frontend/android/app/.../UpdaterPlugin.kt`.
//
// All three calls are no-ops on web (the plugin isn't registered there) —
// the in-app updater UI must guard with isNative() before reaching for any
// of these. On iOS the plugin is currently unregistered too (no iOS build
// yet); planning the same shape when we add it.

import { type PluginListenerHandle,registerPlugin } from '@capacitor/core'

export interface CurrentVersion {
  version_code: number
  version_name: string
  package_name: string
}

export interface DownloadResult {
  path: string
  sha256: string
  size_bytes: number
}

export interface DownloadProgress {
  bytes_loaded: number
  /** -1 when the server didn't send Content-Length. UI should render an
   *  indeterminate spinner in that case rather than a 0/0 progress bar. */
  bytes_total: number
}

export interface UpdaterPlugin {
  /** Reports BuildConfig.VERSION_CODE + VERSION_NAME from the running APK. */
  getCurrentVersion(): Promise<CurrentVersion>

  /**
   * Downloads `url` to the app cache, streaming + verifying SHA-256 against
   * `sha256` (lowercase hex). Throws if the digest doesn't match — caller
   * should NOT retry without re-fetching /api/updates/latest in case the
   * release was repacked.
   *
   * Subscribe to `download_progress` via `addListener` for byte-level
   * progress events (emitted ~every 250ms).
   */
  downloadApk(options: {
    url: string
    sha256: string
    fileName: string
  }): Promise<DownloadResult>

  /**
   * Hands a previously-downloaded APK to the system PackageInstaller via
   * Intent.ACTION_VIEW. Resolves once the activity is launched (NOT once
   * the install succeeds — Android gives no callback for that).
   */
  installApk(options: { path: string }): Promise<void>

  /**
   * Opens the system "Install unknown apps" settings page for this app's
   * package. Use as a fallback when installApk surfaces the
   * "Установка из неизвестных источников" denial dialog.
   */
  openInstallSettings(): Promise<void>

  /** Capacitor's generic listener API — exposed here so the type narrows. */
  addListener(
    eventName: 'download_progress',
    listener: (event: DownloadProgress) => void,
  ): Promise<PluginListenerHandle>
}

export const Updater = registerPlugin<UpdaterPlugin>('Updater')
