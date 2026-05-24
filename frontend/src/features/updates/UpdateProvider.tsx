import { type ReactNode, useCallback, useEffect, useRef, useState } from 'react'

import { fetchLatestRelease } from '../../shared/api/updatesApi'
import { Updater } from '../../shared/nativeUpdater'
import { isNative } from '../../shared/platform'
import { UpdateContext, type UpdateState } from './UpdateContext'

const PREFS_LAST_CHECK = 'update_check_last_at'
const PREFS_DISMISSED_CODE = 'update_dismissed_version_code'

/**
 * Hosts the update state machine + side effects (HTTP poll, native plugin
 * calls, Preferences persistence). Mount once near the top of the React
 * tree — the banner / mandatory screen consume via useUpdate().
 *
 * On web (isNative === false) this provider runs in a degenerate "always
 * up to date" mode: the native plugin isn't registered, so we have no
 * versionCode to compare and no way to install. Banners are silently
 * suppressed.
 */
export function UpdateProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<UpdateState>({ status: 'loading' })
  // Hold the current native versionCode so we can re-derive state.mandatory
  // when the banner is dismissed without re-fetching the release.
  const currentVersionRef = useRef<number | null>(null)
  // Persisted dismissed-version cache. Hydrated in the initial useEffect
  // so the synchronous code path in `checkNow` can apply it without
  // awaiting Preferences for every cold-start check.
  const dismissedVersionRef = useRef<number | null>(null)

  const setStateIfDifferent = useCallback((next: UpdateState) => {
    setState((prev) => (statesEqual(prev, next) ? prev : next))
  }, [])

  const performCheck = useCallback(
    async (force: boolean): Promise<void> => {
      if (!isNative()) {
        // Web path: we never offer updates here. Render children as if up to date.
        setStateIfDifferent({ status: 'up_to_date', currentVersionCode: 0 })
        return
      }

      // Resolve our own versionCode lazily — it doesn't change at runtime
      // but caching it once across re-checks avoids round-tripping the JNI
      // bridge each time.
      if (currentVersionRef.current === null) {
        try {
          const v = await Updater.getCurrentVersion()
          currentVersionRef.current = v.version_code
        } catch (err) {
          setStateIfDifferent({
            status: 'error',
            message: 'cannot read installed version',
            currentVersionCode: null,
            mandatory: false,
          })
          console.warn('getCurrentVersion failed:', err)
          return
        }
      }
      const currentCode = currentVersionRef.current

      // Note: a previous version used a 24h debounce here to skip the
      // HTTP poll on rapid cold starts. That had a bug — when the
      // debounce hit, we set state to 'up_to_date' unconditionally,
      // which hid the banner even when an available release existed in
      // memory. Until we persist the release-row itself across cold
      // starts (and re-evaluate the version comparison from cache), the
      // simpler invariant is to always poll. ~300-byte response is
      // cheap; cold starts are infrequent enough that this isn't load.
      void force // intentionally unused while the debounce is disabled

      let latest
      try {
        latest = await fetchLatestRelease('android')
      } catch (err) {
        // Transport / 5xx — silent fail. Surface as error state so the
        // settings panel can show "Не удалось проверить", but the banner
        // simply doesn't appear.
        setStateIfDifferent({
          status: 'error',
          message: err instanceof Error ? err.message : String(err),
          currentVersionCode: currentCode,
          mandatory: false,
        })
        return
      }
      // Whether or not we found a release, remember the check time so the
      // 24h debounce kicks in. Failed checks are NOT remembered — we want
      // to retry sooner.
      const { Preferences } = await import('@capacitor/preferences')
      await Preferences.set({ key: PREFS_LAST_CHECK, value: String(Date.now()) })

      if (!latest || latest.version_code <= currentCode) {
        setStateIfDifferent({ status: 'up_to_date', currentVersionCode: currentCode })
        return
      }

      const mandatory = currentCode < latest.min_supported_version_code

      // Respect a soft-dismiss (only meaningful for non-mandatory updates).
      if (!mandatory && dismissedVersionRef.current === latest.version_code) {
        setStateIfDifferent({ status: 'up_to_date', currentVersionCode: currentCode })
        return
      }

      setStateIfDifferent({
        status: 'available',
        release: latest,
        currentVersionCode: currentCode,
        mandatory,
      })
    },
    [setStateIfDifferent],
  )

  // Initial cold-start check. Hydrate the dismissed-version cache first so
  // `performCheck` can read it synchronously.
  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (isNative()) {
        try {
          const { Preferences } = await import('@capacitor/preferences')
          const { value } = await Preferences.get({ key: PREFS_DISMISSED_CODE })
          if (value) {
            const code = Number(value)
            if (Number.isFinite(code)) {
              dismissedVersionRef.current = code
            }
          }
        } catch (err) {
          console.warn('hydrate dismissed-version failed:', err)
        }
      }
      if (cancelled) return
      await performCheck(false)
    })()
    return () => {
      cancelled = true
    }
  }, [performCheck])

  const checkNow = useCallback(async () => {
    setState({ status: 'loading' })
    await performCheck(true)
  }, [performCheck])

  const dismiss = useCallback(async () => {
    if (state.status !== 'available' || state.mandatory) return
    const code = state.release.version_code
    dismissedVersionRef.current = code
    const { Preferences } = await import('@capacitor/preferences')
    await Preferences.set({ key: PREFS_DISMISSED_CODE, value: String(code) })
    setState({ status: 'up_to_date', currentVersionCode: state.currentVersionCode })
  }, [state])

  const startDownload = useCallback(async () => {
    if (state.status !== 'available') return
    if (!isNative()) return
    const release = state.release
    setState({
      status: 'downloading',
      release,
      currentVersionCode: state.currentVersionCode,
      mandatory: state.mandatory,
      progress: { loaded: 0, total: release.size_bytes },
    })

    const handle = await Updater.addListener('download_progress', (ev) => {
      setState((prev) =>
        prev.status === 'downloading'
          ? { ...prev, progress: { loaded: ev.bytes_loaded, total: ev.bytes_total } }
          : prev,
      )
    })

    try {
      const result = await Updater.downloadApk({
        url: release.url,
        sha256: release.sha256,
        fileName: `messenger-v${release.version_code}.apk`,
      })
      await handle.remove()
      setState({
        status: 'ready_to_install',
        release,
        path: result.path,
        currentVersionCode: state.currentVersionCode,
        mandatory: state.mandatory,
      })
    } catch (err) {
      await handle.remove()
      setState({
        status: 'error',
        message: err instanceof Error ? err.message : String(err),
        currentVersionCode: state.currentVersionCode,
        mandatory: state.mandatory,
      })
    }
  }, [state])

  // Set when install() is invoked. The next foreground transition compares
  // the installed versionCode against this — if unchanged, the install was
  // rejected (signature mismatch, user denied the system dialog, or unknown-
  // sources permission missing). We surface that as an error so the user
  // actually sees what happened instead of the banner silently reappearing.
  const installAttemptRef = useRef<{
    expectedVersionCode: number
    previousVersionCode: number
    mandatory: boolean
  } | null>(null)

  const install = useCallback(async () => {
    if (state.status !== 'ready_to_install') return
    if (!isNative()) return
    installAttemptRef.current = {
      expectedVersionCode: state.release.version_code,
      previousVersionCode: state.currentVersionCode,
      mandatory: state.mandatory,
    }
    try {
      await Updater.installApk({ path: state.path })
      // Android takes over from here. On success the process is killed and
      // restarted at the new versionCode; on rejection (signature mismatch /
      // user denied / unknown-sources off) the app stays running and we'll
      // detect it via the appStateChange listener below.
    } catch (err) {
      installAttemptRef.current = null
      setState({
        status: 'error',
        message: err instanceof Error ? err.message : String(err),
        currentVersionCode: state.currentVersionCode,
        mandatory: state.mandatory,
      })
    }
  }, [state])

  // Detect "user came back from the Android installer dialog without the
  // version actually changing" — that's how rejected installs manifest
  // (signature mismatch is the most common one when migrating from a
  // sideloaded debug-signed APK to CI-signed releases).
  useEffect(() => {
    if (!isNative()) return
    let cleanup: (() => void) | undefined
    void (async () => {
      try {
        const { App } = await import('@capacitor/app')
        const handle = await App.addListener('appStateChange', (ev) => {
          if (!ev.isActive) return
          const attempt = installAttemptRef.current
          if (!attempt) return
          // Re-read the installed versionCode. If it bumped — Android
          // succeeded and we'd already have been killed; this branch is
          // a defensive no-op. If it didn't bump — install failed.
          void Updater.getCurrentVersion()
            .then((v) => {
              if (v.version_code >= attempt.expectedVersionCode) {
                installAttemptRef.current = null
                return
              }
              installAttemptRef.current = null
              setState({
                status: 'error',
                message:
                  'Установка отклонена системой. Возможные причины: подпись APK не совпадает с установленной версией (тогда удалите приложение и поставьте новый APK с GitHub Releases вручную), либо не разрешена установка из неизвестных источников.',
                currentVersionCode: attempt.previousVersionCode,
                mandatory: attempt.mandatory,
              })
            })
            .catch(() => {
              installAttemptRef.current = null
            })
        })
        cleanup = () => {
          void handle.remove()
        }
      } catch (err) {
        console.warn('failed to subscribe to appStateChange:', err)
      }
    })()
    return () => {
      cleanup?.()
    }
  }, [])

  return (
    <UpdateContext.Provider value={{ state, checkNow, dismiss, startDownload, install }}>
      {children}
    </UpdateContext.Provider>
  )
}

/** Shallow comparison just deep enough to avoid identical-state re-renders. */
function statesEqual(a: UpdateState, b: UpdateState): boolean {
  if (a.status !== b.status) return false
  switch (a.status) {
    case 'loading':
      return true
    case 'up_to_date':
      return a.currentVersionCode === (b as typeof a).currentVersionCode
    case 'available':
      return (
        a.release.version_code === (b as typeof a).release.version_code &&
        a.mandatory === (b as typeof a).mandatory
      )
    case 'downloading':
      return (
        a.release.version_code === (b as typeof a).release.version_code &&
        a.progress.loaded === (b as typeof a).progress.loaded
      )
    case 'ready_to_install':
      return a.path === (b as typeof a).path
    case 'error':
      return a.message === (b as typeof a).message
  }
}
