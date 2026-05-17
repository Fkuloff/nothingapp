import { useState } from 'react'

import { useUpdate } from './UpdateContext'

/**
 * Non-blocking top banner shown on ChatsPage when a NEW non-mandatory
 * release is available. Mandatory releases are rendered by
 * UpdateMandatoryScreen instead and never reach this component.
 *
 * Click "Подробнее" → reveals changelog inline.
 * Click "Позже"   → dismisses for this version code (Preferences-persisted).
 * Click "Обновить" → starts the download flow; banner becomes progress bar.
 */
export function UpdateBanner() {
  const { state, startDownload, install, dismiss } = useUpdate()
  const [expanded, setExpanded] = useState(false)

  // Only render in soft-update flow states. Mandatory + everything outside
  // the update-pending statuses fall through to null.
  if (
    state.status !== 'available' &&
    state.status !== 'downloading' &&
    state.status !== 'ready_to_install'
  ) {
    return null
  }
  // Mandatory updates take over the whole screen via UpdateMandatoryScreen
  // — never let the banner compete for the user's attention there.
  if (state.status === 'available' && state.mandatory) return null
  if (state.status !== 'available' && state.mandatory) return null

  const release = state.release
  const fileSizeMb = (release.size_bytes / (1024 * 1024)).toFixed(1)

  return (
    <div className="update-banner" role="region" aria-label="Доступно обновление">
      <div className="update-banner__row">
        <div className="update-banner__text">
          <span className="update-banner__icon" aria-hidden="true">✨</span>
          <div className="update-banner__copy">
            <span className="update-banner__title">
              Доступно обновление v{release.version_name}
            </span>
            <span className="update-banner__meta">{fileSizeMb} MB</span>
          </div>
        </div>
        <div className="update-banner__actions">
          {state.status === 'available' && (
            <>
              <button
                type="button"
                className="update-banner__btn update-banner__btn--ghost"
                onClick={() => setExpanded((v) => !v)}
              >
                {expanded ? 'Свернуть' : 'Подробнее'}
              </button>
              <button
                type="button"
                className="update-banner__btn update-banner__btn--ghost"
                onClick={() => void dismiss()}
              >
                Позже
              </button>
              <button
                type="button"
                className="update-banner__btn update-banner__btn--primary"
                onClick={() => void startDownload()}
              >
                Обновить
              </button>
            </>
          )}
          {state.status === 'downloading' && <DownloadProgress state={state} />}
          {state.status === 'ready_to_install' && (
            <button
              type="button"
              className="update-banner__btn update-banner__btn--primary"
              onClick={() => void install()}
            >
              Установить
            </button>
          )}
        </div>
      </div>
      {expanded && release.changelog && (
        <pre className="update-banner__changelog">{release.changelog}</pre>
      )}
    </div>
  )
}

function DownloadProgress({
  state,
}: {
  state: Extract<ReturnType<typeof useUpdate>['state'], { status: 'downloading' }>
}) {
  const { loaded, total } = state.progress
  const pct = total > 0 ? Math.min(100, Math.round((loaded / total) * 100)) : null
  return (
    <div className="update-banner__progress" aria-live="polite">
      <span className="update-banner__progress-label">
        {pct === null ? 'Загрузка…' : `Загрузка ${pct}%`}
      </span>
      <div
        className="update-banner__progress-bar"
        role="progressbar"
        aria-valuenow={pct ?? undefined}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className="update-banner__progress-fill"
          style={{ width: pct === null ? '30%' : `${pct}%` }}
        />
      </div>
    </div>
  )
}
