# Messenger

Минималистичный real-time мессенджер с React frontend и Go backend.

## Возможности

- 💬 Личные чаты в реальном времени (WebSocket)
- 🔐 JWT аутентификация
- 📎 Файловые вложения с миниатюрами
- 🟢 Система присутствия (онлайн/оффлайн)
- 📨 Офлайн-сообщения
- ✏️ Редактирование и удаление сообщений
- 💬 Ответы на сообщения (threading)
- 👥 Управление контактами

## Технологии

**Backend:** Go 1.25+ • Gin • PostgreSQL 16 • WebSocket • JWT

**Frontend:** React 18 • TypeScript • Vite • Nginx

**Infrastructure:** Docker • GitHub Actions • GHCR • Trivy

## Быстрый старт

```bash
# Клонировать
git clone https://github.com/your-username/messenger.git
cd messenger

# Настроить .env
cp .env.example .env
nano .env

# Запустить
docker-compose up -d
```

Приложение доступно на http://localhost

## Структура проекта

```
messenger/
├── backend/          # Go backend (Gin + PostgreSQL)
├── frontend/         # React frontend (TypeScript + Vite)
├── .github/          # CI/CD workflows
└── docker-compose.yml
```

## Архитектура

**Clean Architecture:**
```
Handlers → Services → Repositories → Models
```

**WebSocket:**
- Глобальное подключение на пользователя
- Multiplexing по chat_id
- Автоматическая доставка офлайн-сообщений

## API

```
POST /api/auth/register|login
GET  /api/chats
WS   /ws
POST /api/chats/:chatId/messages/:messageId/attachments
GET  /api/presence/:user_id
```

## Разработка

```bash
# Backend
cd backend
go run cmd/server/main.go

# Frontend
cd frontend
npm install && npm run dev

# Docker
docker-compose up -d
```

## CI/CD

GitHub Actions автоматически линтит, тестирует, собирает Docker образы (AMD64/ARM64) и публикует в GHCR.

## Развертывание

```bash
mkdir ~/messenger && cd ~/messenger
# Скопируйте docker-compose.prod.yml и .env.production.example
cp .env.production.example .env
nano .env
docker-compose up -d
```

## Лицензия

MIT
