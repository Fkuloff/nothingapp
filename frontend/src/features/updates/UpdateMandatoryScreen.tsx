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
  const { state, startDownload, install, retry, dismiss } = useUpdate()

  // Error state also has a `mandatory` flag — if a mandatory download fails,
  // we still need to keep blocking the rest of the UI. Previously this state
  // wasn't in the matched set, so the screen disappeared and let the user
  // into the (still-blocked) app; on next cold start performCheck would
  // re-derive mandatory and the loop repeated.
  const mandatory =
    (state.status === 'available' ||
      state.status === 'downloading' ||
      state.status === 'ready_to_install' ||
      state.status === 'error') &&
    state.mandatory

  if (!mandatory) return <>{children}</>

  // Error state: show the message + a Retry button, no release fields available.
  if (state.status === 'error') {
    return (
      <div className="update-mandatory">
        <div className="update-mandatory__card">
          <div className="update-mandatory__icon" aria-hidden="true">⚠️</div>
          <h2 className="update-mandatory__title">Не удалось обновиться</h2>
          <p className="update-mandatory__message" style={{ whiteSpace: 'pre-wrap' }}>
            {state.message || 'неизвестная ошибка'}
          </p>
          <p className="update-mandatory__meta">
            Проверьте подключение к сети и попробуйте снова. Если не помогает —
            скачайте APK вручную:{' '}
            <a
              href="https://github.com/Fkuloff/messenger/releases/latest"
              target="_blank"
              rel="noreferrer noopener"
            >
              GitHub Releases
            </a>
          </p>
          <div className="update-mandatory__action">
            <button
              type="button"
              className="update-mandatory__btn"
              onClick={() => void retry()}
            >
              Повторить
            </button>
            <button
              type="button"
              className="update-mandatory__btn update-mandatory__btn--ghost"
              onClick={() => void dismiss()}
            >
              Остаться на текущей версии
            </button>
          </div>
        </div>
      </div>
    )
  }

  // Now we know we're in one of the mandatory-flagged states. Pull out the
  // release (always present in those states) for display.
  const release = state.release
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
