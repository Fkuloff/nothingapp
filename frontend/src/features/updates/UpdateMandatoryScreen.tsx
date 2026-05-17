import type { ReactNode } from 'react'

import { useUpdate } from './UpdateContext'

/**
 * Full-screen gate shown when the running app is older than the backend's
 * `min_supported_version_code`. Wraps the whole app — if mandatory, the
 * rest of the UI doesn't render. Otherwise renders `children` through.
 *
 * No dismiss button by design. The only way out is to install the update,
 * which kills our process and Android brings up the new version.
 */
export function UpdateMandatoryScreen({ children }: { children: ReactNode }) {
  const { state, startDownload, install } = useUpdate()

  const mandatory =
    (state.status === 'available' ||
      state.status === 'downloading' ||
      state.status === 'ready_to_install') &&
    state.mandatory

  if (!mandatory) return <>{children}</>

  // Now we know we're in one of the mandatory-flagged states. Pull out the
  // release (always present in those states) for display.
  const release =
    state.status === 'available' || state.status === 'downloading' || state.status === 'ready_to_install'
      ? state.release
      : null
  if (!release) return <>{children}</>

  const fileSizeMb = (release.size_bytes / (1024 * 1024)).toFixed(1)

  return (
    <div className="update-mandatory">
      <div className="update-mandatory__card">
        <div className="update-mandatory__icon" aria-hidden="true">🔒</div>
        <h2 className="update-mandatory__title">Требуется обновление</h2>
        <p className="update-mandatory__message">
          Эта версия больше не поддерживается. Установите v{release.version_name},
          чтобы продолжить пользоваться приложением.
        </p>
        <div className="update-mandatory__meta">Размер: {fileSizeMb} MB</div>
        {release.changelog && (
          <pre className="update-mandatory__changelog">{release.changelog}</pre>
        )}
        <div className="update-mandatory__action">
          {state.status === 'available' && (
            <button
              type="button"
              className="update-mandatory__btn"
              onClick={() => void startDownload()}
            >
              Скачать обновление
            </button>
          )}
          {state.status === 'downloading' && (
            <ProgressBar loaded={state.progress.loaded} total={state.progress.total} />
          )}
          {state.status === 'ready_to_install' && (
            <button
              type="button"
              className="update-mandatory__btn"
              onClick={() => void install()}
            >
              Установить
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function ProgressBar({ loaded, total }: { loaded: number; total: number }) {
  const pct = total > 0 ? Math.min(100, Math.round((loaded / total) * 100)) : null
  return (
    <div className="update-mandatory__progress" aria-live="polite">
      <div
        className="update-mandatory__progress-bar"
        role="progressbar"
        aria-valuenow={pct ?? undefined}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className="update-mandatory__progress-fill"
          style={{ width: pct === null ? '40%' : `${pct}%` }}
        />
      </div>
      <span className="update-mandatory__progress-label">
        {pct === null ? 'Загрузка…' : `${pct}%`}
      </span>
    </div>
  )
}
