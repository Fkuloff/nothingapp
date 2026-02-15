# Messenger

Real-time мессенджер с React frontend и Go backend.

## Возможности

- 💬 Личные чаты в реальном времени (WebSocket)
- 🔐 JWT аутентификация
- 🔒 Шифрование сообщений (AES-256-GCM)
- 📎 Файловые вложения через S3/MinIO
- 🔔 Web Push уведомления (VAPID)
- 🟢 Система присутствия (онлайн/оффлайн)
- 📨 Офлайн-доставка сообщений
- ✏️ Редактирование и удаление сообщений
- 💬 Ответы на сообщения (threading)
- 👥 Управление контактами
- 🌗 Темная/светлая тема

## Технологии

**Backend:** Go 1.24 · Gin · GORM · PostgreSQL 16 · WebSocket · JWT · zap

**Frontend:** React 19 · TypeScript · Vite · Bootstrap 5 · React Router 7

**Infrastructure:** Docker · MinIO · Nginx · Certbot · GitHub Actions · GHCR · Distroless

## Быстрый старт

```bash
# Клонировать
git clone https://github.com/fkuloff/messenger.git
cd messenger

# Настроить .env
cp .env.example .env
nano .env

# Запустить (postgres + minio + backend + frontend)
docker-compose up --build
```

Приложение доступно на `http://localhost`

### Минимальные переменные

```bash
POSTGRES_PASSWORD=your_password
JWT_SECRET=your_very_secure_jwt_secret_at_least_32_chars
MESSAGE_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

Остальные имеют значения по умолчанию. Полный список — в `.env.example`.

## Структура проекта

```
messenger/
├── backend/                  # Go backend (Gin + GORM + PostgreSQL)
│   ├── cmd/server/           # Точка входа
│   └── internal/             # Handlers, services, repos, models, crypto
├── frontend/                 # React 19 (TypeScript + Vite + Bootstrap)
│   └── src/                  # Pages, features, shared
├── nginx-proxy/              # Production nginx конфиг
├── .github/workflows/        # CI/CD pipeline
├── docker-compose.yml        # Dev: postgres, minio, backend, frontend
└── docker-compose.prod.yml   # Prod: + nginx-proxy, SSL (Certbot)
```

## Архитектура

**Backend — Clean Architecture:**
```
Handlers → Services → Repositories → Models
```

**WebSocket:**
- Глобальное подключение `/ws`, мультиплексирование по `chat_id`
- Worker pool (50 воркеров) для broadcast
- Heartbeat (ping/pong 54s), rate limit 10 msg/s
- Офлайн-очередь → доставка при переподключении

**Шифрование:**
- AES-256-GCM encryption at rest
- Уникальный 12-byte nonce для каждого сообщения
- Ключ: `MESSAGE_ENCRYPTION_KEY` (base64, 32 байта)

**Хранилище файлов:**
- S3-совместимое (MinIO) с presigned URLs
- Dual endpoint: internal (Docker) + public (browser)

## API

**Public:**
```
POST   /api/auth/register
POST   /api/auth/login
GET    /health
GET    /api/avatars/:user_id
GET    /api/attachments/:id
```

**Protected** (JWT):
```
POST   /api/auth/logout           GET  /api/auth/me
GET    /api/chats                 POST /api/chats
GET    /api/chats/:id             DELETE /api/chats/:id
POST   /api/chats/:id/clear       GET  /api/chats/:id/messages
POST   /api/chats/:id/messages/:message_id/attachments
DELETE /api/attachments/:id
GET    /api/profile               PUT  /api/profile
GET    /api/contacts              POST /api/contacts/:user_id
DELETE /api/contacts/:user_id     GET  /api/users/search
POST   /api/user/avatar           DELETE /api/user/avatar
GET    /api/presence/:user_id
GET    /api/unread                GET  /api/unread/counts
GET    /api/push/vapid-key        POST /api/push/subscribe
POST   /api/push/unsubscribe      GET  /api/push/status
WS     /ws
```

## Разработка

```bash
# Docker (рекомендуется)
docker-compose up --build              # Все сервисы
docker-compose --profile dev up        # + pgAdmin

# Backend
cd backend
go run cmd/server/main.go
go test -v -race ./...

# Frontend
cd frontend
npm install
npm run dev                            # Vite dev server
npm run build                          # Production build
npm run lint                           # ESLint
```

## CI/CD

GitHub Actions pipeline (`main`):

1. **Lint** — golangci-lint (backend) + ESLint (frontend)
2. **Test** — `go test -race -coverprofile` с PostgreSQL
3. **Build** — бинарник Go + `npm run build`
4. **Docker** — сборка и push в GHCR (linux/amd64)
5. **Deploy** — SSH deploy на сервер, `docker compose pull && up -d`

Триггер: push/PR в `main`. Docker push + deploy только при push.

## Развертывание (Production)

```bash
# На сервере
mkdir ~/messenger && cd ~/messenger
# Скопировать docker-compose.prod.yml, nginx-proxy/, .env
cp .env.example .env
nano .env

# Первый запуск (SSL сертификат)
chmod +x init-letsencrypt.sh && ./init-letsencrypt.sh

# Запуск
docker compose -f docker-compose.prod.yml up -d
```

Production стек: Nginx (reverse proxy + SSL) → Frontend (Nginx) → Backend (Distroless) → PostgreSQL + MinIO.

## Ограничения

| Параметр | Лимит |
|----------|-------|
| WebSocket сообщение | 10 KB |
| Соединений на пользователя | 5 |
| Rate limit (WS) | 10 msg/s |
| Rate limit (HTTP) | 30 req/s/IP |
| Изображения | 100 MB |
| Видео | 500 MB |
| Документы | 50 MB |
| Файлов в сообщении | 10 |

## Лицензия

MIT
