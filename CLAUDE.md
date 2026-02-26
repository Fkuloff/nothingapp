# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

Real-time messenger (Go + React) with WebSocket chat, JWT auth, E2E message encryption (AES-256-GCM), file attachments via S3/MinIO, web push notifications, 1-on-1 audio calls (WebRTC), and contact management.

## Project Structure

```
messenger/
├── backend/          # Go backend (Gin, GORM, PostgreSQL)
│   ├── cmd/server/   # Entry point
│   └── internal/     # Handlers, services, repos, models
├── frontend/         # React 19 + TypeScript + Vite + Bootstrap
│   └── src/          # Pages, features, shared components
├── nginx-proxy/      # Production nginx config
├── docker-compose.yml        # Dev: postgres, minio, backend, frontend
└── docker-compose.prod.yml   # Prod: adds nginx-proxy, SSL
```

## Build and Run

### Prerequisites
- Go 1.24.2+
- Node.js (for frontend)
- PostgreSQL + MinIO (or use docker-compose)

### Development (Docker - recommended)
```bash
docker-compose up --build              # Start all services
docker-compose --profile dev up        # Include pgAdmin
docker-compose down                    # Stop
```

### Backend only
```bash
cd backend
go run cmd/server/main.go              # Run server
go build -o messenger cmd/server/main.go  # Build
```

### Frontend only
```bash
cd frontend
npm install
npm run dev          # Vite dev server
npm run build        # Production build (tsc + vite)
npm run lint         # ESLint
npm run lint:fix     # ESLint autofix (imports sorting, unused imports)
npm run knip         # Find unused files, exports, dependencies
```

### Tests
```bash
cd backend && go test ./...            # Run all tests
```

Existing test files:
- `backend/internal/config/config_test.go`
- `backend/internal/handlers/middleware_test.go`
- `backend/internal/models/chat_test.go`, `user_test.go`
- `backend/internal/services/file_validator_test.go`, `presence_service_test.go`

## Configuration

### Required environment variables
- `DB_URL` — PostgreSQL connection string
- `JWT_SECRET` — Min 32 chars, no weak patterns (enforced in `backend/internal/config/config.go`)
- `MESSAGE_ENCRYPTION_KEY` — Base64-encoded 32-byte key for AES-256-GCM. Generate: `openssl rand -base64 32`. Changing this makes existing messages unreadable.

### S3/MinIO storage (required)
- `STORAGE_S3_ENDPOINT` — Internal S3 endpoint (e.g. `http://minio:9000`)
- `STORAGE_S3_PUBLIC_ENDPOINT` — Public endpoint for presigned URLs
- `STORAGE_S3_BUCKET`, `STORAGE_S3_REGION`, `STORAGE_S3_ACCESS_KEY`, `STORAGE_S3_SECRET_KEY`
- `STORAGE_S3_USE_SSL`, `STORAGE_S3_PRESIGNED_EXPIRY`

### Optional
- `PORT` — Server port (default: 8080)
- `ALLOWED_ORIGINS` — Comma-separated CORS origins
- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT` — Web push (graceful degradation if unset)

See `.env.example` for full reference.

## Architecture

### Backend — Layered Architecture

```
Handlers → Services → Repositories → Models
```

1. **Handlers** (`backend/internal/handlers/`) — HTTP/WebSocket request handling
2. **Services** (`backend/internal/services/`) — Business logic
3. **Repositories** (`backend/internal/repositories/`) — Database operations (GORM)
4. **Models** (`backend/internal/models/`) — Data structures

**Rules**: Handlers MUST NOT access repositories directly. Services encapsulate all business logic and data access.

**Key handler files**:
- `routes.go` — All route registration
- `websocket.go` — Global WebSocket handler
- `websocket_actions.go` — WS message actions (send/edit/delete/mark_read)
- `websocket_call.go` — WebRTC call signaling relay (offer/answer/ICE/hangup/reject)
- `types.go` — Shared type definitions
- `constants.go` — Limits, timeouts
- `response.go` — JSON response helpers
- `middleware.go` — JWT middleware

**Other important packages**:
- `backend/internal/app/` — Application bootstrap and dependency injection
- `backend/internal/crypto/` — AES-256-GCM message encryption at rest
- `backend/internal/logger/` — Structured logging (zap)
- `backend/internal/shutdown/` — Graceful shutdown (SIGTERM/SIGINT, 30s timeout)
- `backend/internal/storage/` — S3 storage abstraction (presigned URLs)
- `backend/internal/config/` — Configuration loading and validation
- `backend/internal/testutil/` — Test fixtures and DB helpers

### Frontend — React 19 + TypeScript

```
frontend/src/
├── pages/      # LoginPage, RegisterPage, ChatsPage, ProfilePage, ContactsPage, SettingsPage
├── features/   # auth, calls, chats, contacts, menu, profile, settings
├── shared/     # api, components, context (Auth, Theme), hooks, utils, constants
├── router/     # React Router setup
└── App.tsx     # Root component
```

Stack: React 19, TypeScript, Vite 7, Bootstrap 5, React Router 7.

**Shared components** (`frontend/src/shared/components/`):
- `Icons.tsx` — Reusable SVG icons (`CloseIcon`, `ChatBubbleIcon`, `PhoneIcon`, `MicIcon`)
- `Toast.tsx` / `ToastContext.tsx` — Toast notification system
- `PushToggle.tsx` — Push notification toggle

**Shared hooks** (`frontend/src/shared/hooks/`):
- `useConfirmAction(timeout?)` — Inline confirmation with auto-reset (used in contacts, profile)
- `useGlobalWebSocket()` — Global WS connection management
- `useModalBehavior()` — Modal open/close, click-outside, escape-key
- `usePushNotifications()` — Push subscription management
- `useTheme()` — Theme context consumer

**Frontend linting**: ESLint 9 flat config with `typescript-eslint/strict`, `unused-imports` (autofix), `simple-import-sort` (autofix). Run `npm run knip` to find dead exports/files.

### WebSocket Architecture

Single global WebSocket at `/ws` handles all chats (multiplexed by `chat_id`):

- **Connection map**: `clients map[uint][]*wsClient` — userID → connections
- **Worker pool**: 50 workers for broadcast concurrency
- **Presence**: In-memory online/offline tracking with heartbeat (ping/pong every 54s)
- **Offline delivery**: Messages saved to `unread_messages` table, delivered on reconnect
- **Actions**: `send`, `edit`, `delete`, `mark_read`, `call_offer`, `call_answer`, `call_ice`, `call_hangup`, `call_reject`

### Message Encryption

Server-side AES-256-GCM encryption at rest (`backend/internal/crypto/crypto.go`):
- Messages encrypted before DB write, decrypted on read
- Each message gets a unique random 12-byte nonce (IV)
- IV stored alongside ciphertext in the database

### Storage

S3-only via MinIO (`backend/internal/storage/s3_storage.go`):
- Presigned URLs for secure browser access (configurable expiry)
- Dual endpoint support: internal (Docker network) + public (browser)
- Factory pattern in `backend/internal/storage/factory.go`

### Audio Calling (WebRTC)

1-on-1 audio calls via WebRTC with signaling over the existing WebSocket:

- **Backend** (`websocket_call.go`): Stateless relay — validates chat access, checks 1-on-1 only, forwards signaling to the other user via `broadcastToUser`. No call state stored in DB/memory.
- **Frontend** (`frontend/src/features/calls/`):
  - `useWebRTC.ts` — RTCPeerConnection lifecycle, media streams, trickle ICE buffering
  - `CallContext.tsx` — Context + `useCallContext` hook (split from provider for react-refresh lint)
  - `CallProvider.tsx` — Call state machine (idle → outgoing/incoming → active), mounted in `main.tsx` above router so calls survive navigation
  - `IncomingCallModal.tsx` — Modal with accept/reject buttons
  - `ActiveCallOverlay.tsx` — Fixed top bar with timer, mute, hangup
- **STUN only**: `stun.l.google.com:19302` — no TURN server for now
- **registerSend pattern**: `CallProvider` needs WebSocket `send()` which lives in `ChatsPage`. ChatsPage registers its send function via `callContext.registerSend(send)` on mount.

## API Routes

**Public**: `POST /api/auth/register`, `POST /api/auth/login`, `GET /health`, `GET /api/avatars/:user_id`, `GET /api/attachments/:id`

**Protected** (JWT required):
- Auth: `POST /api/auth/logout`, `GET /api/auth/me`
- Chats: `GET|POST /api/chats`, `GET|DELETE /api/chats/:id`, `POST /api/chats/:id/clear`, `GET /api/chats/:id/messages`
- Attachments: `POST /api/chats/:id/messages/:message_id/attachments`, `DELETE /api/attachments/:id`
- Profile: `GET|PUT /api/profile`, `GET /api/profile/:user_id`
- Contacts: `GET /api/contacts`, `POST|DELETE /api/contacts/:user_id`
- Users: `GET /api/users/search`, `POST /api/user/avatar`, `DELETE /api/user/avatar`
- Presence: `GET /api/presence/:user_id`
- Unread: `GET /api/unread`, `GET /api/unread/counts`
- Push: `GET /api/push/vapid-key`, `POST /api/push/subscribe`, `POST /api/push/unsubscribe`, `GET /api/push/status`
- WebSocket: `GET /ws`

## Database Models

Key relationships:
- **User** ↔ **Chat**: One-on-one via `User1ID`/`User2ID`
- **Chat** → **Messages** → **Attachments**: One-to-many chains
- **Message** → **ReplyTo**: Self-referential for threading
- **User** ↔ **Contacts**: Many-to-many via Contact table
- **User** → **UnreadMessage**: Offline message queue
- **User** → **PushSubscription**: Web push subscriptions

Soft deletes: Messages use `IsDeleted` flag to preserve reply chains.

## Constraints

- WebSocket: 64KB max message (for SDP payloads), 5 connections/user, 10 msg/s rate limit
- Files: Images 100MB, Videos 500MB, Docs 50MB, max 10 per message
- JWT: ≥32 char secret, validated issuer/audience/JTI
- Rate limit: 30 req/s per IP (global)

## Common Patterns

### Adding a WebSocket action
1. Add handler in `backend/internal/handlers/websocket_actions.go`
2. Include `action` and `chat_id` fields in message struct
3. Implement logic in service layer, save via repository
4. Check presence → broadcast if online, save to `unread_messages` if offline

### Adding an API endpoint
1. Create handler in `backend/internal/handlers/`
2. Register route in `backend/internal/handlers/routes.go`
3. Use `c.Get("user_id")` for authenticated user ID

### Adding an inline confirmation (frontend)
1. Import `useConfirmAction` from `shared/hooks/useConfirmAction`
2. Destructure `{ confirming, startConfirm, cancelConfirm }`
3. Toggle UI between normal state and confirm buttons on `confirming`
4. Auto-resets after 5s (configurable via `timeout` param)

### Adding icons (frontend)
Use shared icons from `shared/components/Icons.tsx` (`CloseIcon`, `ChatBubbleIcon`).
Do NOT create inline SVGs — add new icons to `Icons.tsx` instead.
