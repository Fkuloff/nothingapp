# In-app self-update: design + behaviour

The mobile app distributes outside Google Play (sideload APK). To save users
from manually grabbing each new build, the app polls a release registry on
startup and offers to download + install newer versions. Critical security
fixes can be pushed as **mandatory** updates that block the UI until the
user installs them.

This doc covers the v1 implementation (banner + mandatory screen, no
hamburger-dot, no settings panel, no instant WS push). Future iterations
that add those layers will append here.

## High-level flow

```
device                              backend                              CI / admin

cold start
   │
   ├──▶ GET /api/updates/latest ───▶ app_releases table
   │       (debounced 24h            └── latest row per platform
   │        unless `force=true`)         (highest version_code)
   │
   ◀──── 200 {version_code, url, sha256, ...}  or  204 No Content
   │
   ├── if latest.version_code <= current → done
   ├── if current < min_supported       → MandatoryScreen
   └── else                              → UpdateBanner (soft)

user taps "Обновить" / "Скачать"
   │
   ├──▶ Updater.downloadApk(url, sha256)
   │       └── streams to cache/updates/<name>.apk
   │       └── verifies SHA-256 in flight
   │       └── emits download_progress events every 250ms
   │
   ├──▶ Updater.installApk(path)
   │       └── FileProvider URI
   │       └── Intent.ACTION_VIEW + application/vnd.android.package-archive
   │       └── system PackageInstaller takes over
   │
   ◀──── (our process dies; user re-launches the new APK)
```

## Components

| Slice | File | Lines |
|---|---|---|
| DB model       | `backend/internal/models/app_release.go`               | ~30  |
| Repository     | `backend/internal/repositories/app_release_repo.go`    | ~55  |
| Service        | `backend/internal/services/app_release_service.go`     | ~150 |
| HTTP handler   | `backend/internal/handlers/app_release.go`             | ~110 |
| Config field   | `backend/internal/config/config.go`                    | +15  |
| Routes wiring  | `backend/internal/handlers/routes.go`                  | +10  |
| Native plugin  | `frontend/android/.../UpdaterPlugin.kt`                | ~180 |
| Manifest perm  | `frontend/android/.../AndroidManifest.xml`             | +10  |
| FileProvider   | `frontend/android/.../res/xml/file_paths.xml`          | +5   |
| TS wrapper     | `frontend/src/shared/nativeUpdater.ts`                 | ~55  |
| API client     | `frontend/src/shared/api/updatesApi.ts`                | ~30  |
| State machine  | `frontend/src/features/updates/UpdateContext.ts`       | ~55  |
| Provider       | `frontend/src/features/updates/UpdateProvider.tsx`     | ~210 |
| Soft banner    | `frontend/src/features/updates/UpdateBanner.tsx`       | ~110 |
| Mandatory UI   | `frontend/src/features/updates/UpdateMandatoryScreen.tsx` | ~95 |
| CSS            | `frontend/src/App.css`                                 | +220 |
| Wiring         | `frontend/src/main.tsx`, `pages/ChatsPage.tsx`         | +15  |

## API

### `GET /api/updates/latest?platform=android` (public)

Returns the highest-`version_code` release row for the platform, or 204 if
none exist.

```json
{
  "ID": 1,
  "platform": "android",
  "version_name": "1.2.3",
  "version_code": 42,
  "min_supported_version_code": 38,
  "url": "https://files.example.com/releases/app-v42.apk",
  "sha256": "abc123...64hex",
  "size_bytes": 6291456,
  "changelog": "## v1.2.3\n- Fix attachment download\n- ...",
  "released_at": "2026-05-17T10:00:00Z"
}
```

Public + unauthenticated by design — a client locked out by a mandatory
update must be able to discover its way out before logging in.

Cache-Control: `no-store`. Response is small (~300 bytes), staleness is
worse than the extra round-trip.

### `POST /api/admin/releases` (admin-only)

Registers a new release row. Header auth via `X-Admin-Key` compared with
`subtle.ConstantTimeCompare` against the `ADMIN_API_KEY` env. Disabled
(returns 503) when the env is unset.

Body matches `services.CreateReleaseRequest`:

```bash
curl -X POST https://api.example.com/api/admin/releases \
  -H "X-Admin-Key: $ADMIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "android",
    "version_name": "1.2.3",
    "version_code": 42,
    "min_supported_version_code": 38,
    "url": "https://files.example.com/releases/app-v42.apk",
    "sha256": "abc123...64hex",
    "size_bytes": 6291456,
    "changelog": "## v1.2.3\n- Fix attachment download"
  }'
```

Validation (service layer):
- `version_name`: 1..32 chars
- `version_code`: positive int
- `min_supported_version_code`: in `(0, version_code]`
- `url`: must be http(s)://
- `sha256`: 64 lowercase hex chars
- `size_bytes`: positive
- `changelog`: ≤ 8 KB
- `(platform, version_code)` must be unique → 409 on conflict

## Native plugin (Kotlin)

`UpdaterPlugin` exposes three methods:

| Method | What it does |
|---|---|
| `getCurrentVersion()` | Returns `BuildConfig.VERSION_CODE` + `VERSION_NAME`. Compared JS-side against the backend's `latest`. |
| `downloadApk(url, sha256, fileName)` | Streams URL → `cache/updates/<name>.apk` in 64KB chunks, updating SHA-256 in flight. Rejects on digest mismatch. Emits `download_progress` event every 250ms. |
| `installApk(path)` | Builds a FileProvider URI for the APK, fires `Intent.ACTION_VIEW` with `application/vnd.android.package-archive`. Android's PackageInstaller takes over from here. |
| `openInstallSettings()` | Deep-links to "Install unknown apps" toggle for this app's package. Used when the user denies the install permission. |

Out-of-process digest check is fine because Android refuses to install any
APK whose signature doesn't match the currently-installed app — so even a
compromised backend can't push a malicious build through this path. The
in-flight SHA-256 just covers the "MITM corrupts the bytes mid-transfer"
case where Android would otherwise fail the install with a corrupt-file
error instead of our clearer toast.

## Permissions

`AndroidManifest.xml` adds `REQUEST_INSTALL_PACKAGES`. On Android 8+ this
is a **special permission** — user must toggle "Install unknown apps" for
the app's package in system settings. We can deep-link them to that
toggle screen via `Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES`, but
the actual toggle is the user's. One-time annoyance per device.

## Frontend state machine

See `frontend/src/features/updates/UpdateContext.ts`. The provider is the
single source of truth; banner + mandatory screen are pure consumers.

```
loading ─▶ up_to_date
        ─▶ available (mandatory: bool)
        ─▶ error

available ─▶ downloading ─▶ ready_to_install
                         ─▶ error

ready_to_install ─▶ (Android takes over)
```

Persistence:
- `update_check_last_at` — ms-since-epoch of last successful poll.
   24h debounce; force-bypassed when caller passes `force=true`.
- `update_dismissed_version_code` — version code the user soft-dismissed.
   Banner stays hidden for that exact version_code; reappears at v+1.

Mandatory updates cannot be dismissed — the screen has no close button.

## Mounting

```
<UpdateProvider>          ← state machine + side effects
  <UpdateMandatoryScreen> ← full-screen gate for force-updates
    <AppRouter>           ← regular routes (login, chats, profile, ...)
      <ChatsPage>
        <UpdateBanner />  ← soft-update strip above the chat layout
        ...
```

The provider runs the initial check on cold start. On web, it short-circuits
to `up_to_date` immediately (no native plugin = no version compare = no
update offered).

## Known limitations

| Case | Behaviour |
|---|---|
| Network drops mid-download | Cache file deleted, state → error. User retries from the banner. |
| SHA-256 mismatch          | Cache file deleted, state → error with explanation. Almost always means CI bug or admin POSTed wrong hash. |
| User denies "Install unknown apps" | Android shows its own settings prompt. Our `openInstallSettings()` helper deep-links them once the banner re-fires. |
| Same versionCode after install | After the system PackageInstaller returns, our process is killed. Next launch reads fresh `getCurrentVersion()` — if the new install actually succeeded, the banner is gone. If the user cancelled, the next debounced check re-offers. |
| Cellular metered network | No special handling in v1. APKs are ~6 MB which is acceptable on metered data. |
| Server returns 5xx        | Silent fail — state → error, banner doesn't appear. Next 24h-debounced retry. |
| Downgrade attempt          | Backend filter (`version_code DESC LIMIT 1`) naturally excludes older rows. Even if surfaced, Android rejects downgrades with `INSTALL_FAILED_VERSION_DOWNGRADE`. |
| Signing key mismatch      | Android rejects with `INSTALL_FAILED_UPDATE_INCOMPATIBLE`. All releases MUST be signed by the same release keystore — losing the key means users have to uninstall + reinstall (losing local E2E vault). |
| Live updates / partial JS rollout | Not implemented in v1. Bundle changes ride with full APK releases. |

## What's deliberately not here

These were considered and skipped for v1 to keep the surface area small.
Re-adding any of them is straightforward.

- **Hamburger menu red dot** — would tap into the menu's badge state.
- **Settings → Обновления панель** — manual "Проверить" button + current
  version + changelog of latest. The provider already exposes `checkNow()`.
- **WS-driven instant refresh** — broadcast a `release_available` event
  from the admin POST handler so online users see the banner in seconds,
  not within 24h. ~45 lines of Go + ~30 of TS.
- **Push notifications** — system tray "Update available v1.2.3". Common
  antipattern; users disable push entirely once we abuse it.

## Operational notes

- No new migrations needed beyond the AutoMigrate of `app_releases`.
- `ADMIN_API_KEY` env var is optional — when unset, admin endpoint
  returns 503 instead of 401. Avoid empty-string bypass on accident.
- Release keystore generation + CI signing pipeline is NOT part of this
  PR. The plugin will happily download a release-signed APK, but until
  we sign one, every install attempt fails with the signature mismatch
  error. v1 is the foundation; the keystore + GitHub Action workflow
  is the next-step.
- Cache directory cleanup: every new `downloadApk` call wipes
  `cache/updates/` first so abandoned partial downloads don't accumulate.
