# Messenger

Современный REST API мессенджер на Go с WebSocket, JWT-аутентификацией, системой присутствия и управлением вложениями.

## Возможности

### Основные функции
- 🔐 **Аутентификация**: Регистрация/логин с JWT токенами (cookie-based)
- 💬 **Личные чаты**: Сообщения в реальном времени через WebSocket
- 👥 **Контакты**: Управление списком контактов, поиск пользователей
- 📎 **Вложения**: Загрузка файлов с автогенерацией миниатюр для изображений
- 🟢 **Presence**: Отслеживание статуса онлайн/оффлайн пользователей
- 📨 **Офлайн-сообщения**: Автоматическая доставка сообщений при подключении
- ✏️ **Редактирование**: Редактирование и удаление отправленных сообщений
- 💬 **Ответы**: Возможность отвечать на конкретные сообщения (threading)

### Технические особенности
- Clean Architecture (Handlers → Services → Repositories)
- Graceful shutdown с таймаутами
- Rate limiting (глобальный + per-connection WebSocket)
- CORS с настраиваемыми origin
- Структурированное логирование (zap)
- SQL-миграции для управления схемой БД
- Расширяемая система хранения файлов (Local/S3)

## Требования

- **Go** >= 1.25.1
- **PostgreSQL** >= 14
- **Docker** (опционально, для локальной БД)

## Быстрый старт

### 1. Настройка окружения

Скопируйте `.env.example` в `.env` и заполните переменные:

```env
# База данных
DB_URL=postgres://postgres:postgres@localhost:5432/messenger?sslmode=disable

# JWT (минимум 32 символа)
JWT_SECRET=your-super-secret-key-at-least-32-characters-long

# Сервер
PORT=8080
ALLOWED_ORIGINS=http://localhost:8080,http://localhost:3000

# Хранилище файлов
STORAGE_TYPE=local
STORAGE_LOCAL_PATH=./uploads
STORAGE_LOCAL_URL=http://localhost:8080/api/files
```

### 2. Запуск PostgreSQL

```bash
docker run -d --name messenger-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=messenger \
  -p 5432:5432 \
  postgres:16
```

### 3. Миграции БД

```bash
# Windows
scripts\migrate.bat up

# Linux/Mac
./scripts/migrate.sh up

# Или напрямую
go run ./cmd/migrate -action=up
```

### 4. Запуск сервера

```bash
go mod tidy
go run ./cmd/server
```

Сервер запустится на `http://localhost:8080`

## API Документация

### Authentication

**POST** `/api/auth/register`
```json
{
  "username": "john_doe",
  "password": "secure_password",
  "name": "John Doe",
  "phone": "+1234567890"
}
```

**POST** `/api/auth/login`
```json
{
  "username": "john_doe",
  "password": "secure_password"
}
```

**GET** `/api/auth/me` - Получить текущего пользователя

**POST** `/api/auth/logout` - Выйти из системы

### Чаты

**GET** `/api/chats` - Список всех чатов с preview последнего сообщения

**POST** `/api/chats` - Создать/получить чат с пользователем
```json
{
  "username": "john_doe"
}
```

**GET** `/api/chats/:id` - Информация о чате

**GET** `/api/chats/:id/messages?limit=100&offset=0` - История сообщений

### WebSocket

**WS** `/ws` - Глобальное WebSocket подключение (одно на пользователя)

Формат сообщений:
```json
{
  "action": "send",
  "chat_id": 1,
  "text": "Hello!",
  "reply_to_id": 5  // опционально
}
```

Доступные action:
- `send` - Отправить сообщение
- `edit` - Редактировать (требуется `message_id`)
- `delete` - Удалить (требуется `message_id`)
- `mark_read` - Пометить чат как прочитанный

### Вложения

**POST** `/api/chats/:chatId/messages/:messageId/attachments`
- Multipart форма с полем `files[]`
- Максимум 10 файлов на сообщение
- Поддержка: изображения (100MB), видео (500MB), документы (50MB)

**GET** `/api/attachments/:id` - Скачать вложение

**GET** `/api/attachments/:id/thumbnail` - Получить миниатюру (для изображений)

**DELETE** `/api/attachments/:id` - Удалить вложение

### Контакты и пользователи

**GET** `/api/contacts` - Список контактов

**POST** `/api/contacts/add/:user_id` - Добавить в контакты

**POST** `/api/contacts/remove/:user_id` - Удалить из контактов

**GET** `/api/users/search?q=john` - Поиск пользователей (мин. 2 символа)

**GET** `/api/profile` - Свой профиль

**GET** `/api/profile/:user_id` - Профиль пользователя

### Presence (статус онлайн)

**GET** `/api/presence/:user_id` - Проверить онлайн-статус
```json
{
  "user_id": 1,
  "is_online": true,
  "last_seen": "2024-01-15T10:30:00Z"
}
```

### Непрочитанные сообщения

**GET** `/api/unread` - Все непрочитанные сообщения

**GET** `/api/unread/counts` - Счетчики непрочитанных по чатам
```json
{
  "unread_counts": {
    "1": 5,
    "2": 12
  }
}
```

### Аватары

**POST** `/api/user/avatar` - Загрузить аватар (multipart/form-data)

**DELETE** `/api/user/avatar` - Удалить аватар

## Архитектура

Проект следует **Clean Architecture** принципам с чёткими границами между слоями:

```
┌─────────────┐
│  Handlers   │  HTTP/WebSocket обработчики, валидация запросов
└──────┬──────┘
       │
┌──────▼──────┐
│  Services   │  Бизнес-логика, оркестрация
└──────┬──────┘
       │
┌──────▼──────┐
│Repositories │  Операции с БД (GORM)
└──────┬──────┘
       │
┌──────▼──────┐
│   Models    │  GORM модели, структуры данных
└─────────────┘
```

### Структура проекта

```
├── cmd/                 # Точки входа приложения (server, migrate)
├── internal/
│   ├── app/             # Инициализация, middleware, роутинг
│   ├── config/          # Конфигурация (.env)
│   ├── handlers/        # HTTP/WebSocket обработчики
│   ├── services/        # Бизнес-логика
│   ├── repositories/    # Data access layer (GORM)
│   ├── models/          # Структуры данных и модели БД
│   ├── storage/         # Абстракция хранилища файлов
│   └── shutdown/        # Graceful shutdown
├── migrations/          # SQL миграции
├── scripts/             # Утилиты (migrate.sh/bat)
└── uploads/             # Локальное хранилище файлов
```

## Ключевые особенности реализации

### WebSocket архитектура

- **Глобальное подключение**: Одно WebSocket соединение на пользователя (не на чат)
- **Multiplexing**: Все сообщения содержат `chat_id` для маршрутизации
- **Presence tracking**: Автоматическое отслеживание через ping/pong (каждые 54 секунды)
- **Offline queue**: Сообщения сохраняются в `unread_messages` если получатель offline
- **Auto-delivery**: При подключении автоматически доставляются все накопленные сообщения

### Система присутствия (PresenceService)

- In-memory хранение статусов
- Автоматическая очистка:
  - 2 минуты без активности → пользователь считается offline
  - 1 час offline → запись удаляется из памяти (предотвращение утечки памяти)
- Поддержка множественных подключений одного пользователя
- Callback система для уведомлений о смене статуса

### Безопасность

- **JWT аутентификация**: Cookie-based с HttpOnly флагом
- **Rate limiting**:
  - Глобальный: 30 req/sec на IP
  - WebSocket: 10 msg/sec на соединение
- **File validation**: Проверка типов, размеров, имен файлов
- **Access control**: Проверка прав на каждую операцию с чатом/сообщением
- **CORS**: Настраиваемый список разрешенных origins

### Миграции базы данных

```bash
# Применить все миграции
scripts/migrate.sh up

# Откатить последнюю
scripts/migrate.sh down

# Проверить статус
scripts/migrate.sh status

# Создать новую миграцию
scripts/migrate.sh create add_feature_name
```

Миграции находятся в `migrations/`. Формат: `NNNNNN_description.up.sql` и `NNNNNN_description.down.sql`.

**Важно**: В production рекомендуется использовать SQL-миграции и отключить GORM AutoMigrate.

## Команды разработки

```bash
# Запуск сервера
go run ./cmd/server

# Сборка
go build -o server.exe ./cmd/server

# Запуск тестов (когда будут добавлены)
go test ./...

# Линтер
golangci-lint run

# Форматирование
go fmt ./...

# Все проверки (make)
make lint
make build
```

## Docker

```bash
# Сборка образа
docker build -t messenger-app .

# Запуск
docker run -p 8080:8080 --env-file .env messenger-app
```

## Расширение функционала

### Добавление нового WebSocket action

1. Добавьте handler в `internal/handlers/websocket_actions.go`:
```go
func (h *WebSocketHandler) handleReaction(userID uint, msgData MessageAction) error {
    // 1. Проверка доступа к чату
    chat, err := h.chatService.FindChatByIDLight(msgData.ChatID)
    if err != nil || !chat.HasUser(userID) {
        return &wsError{message: "Access denied"}
    }

    // 2. Бизнес-логика через сервис
    // ...

    // 3. Broadcast всем участникам чата
    h.broadcastToChat(msgData.ChatID, responseJSON)
    return nil
}
```

2. Зарегистрируйте action в `HandleWebSocket`:
```go
case "reaction":
    handlerErr = h.handleReaction(userID, msgData)
```

### Добавление нового REST endpoint

1. Создайте метод в handler (`internal/handlers/`)
2. Добавьте route в `internal/handlers/routes.go`
3. JWT middleware применяется глобально (кроме `/login`, `/register`)

## Конфигурация

Все настройки задаются через переменные окружения в `.env`:

| Переменная | Описание | Пример |
|-----------|----------|--------|
| `DB_URL` | PostgreSQL connection string | `postgres://user:pass@localhost/db` |
| `JWT_SECRET` | Ключ для JWT (≥32 символа) | `your-secret-key-...` |
| `PORT` | Порт сервера | `8080` |
| `ALLOWED_ORIGINS` | CORS origins (через запятую) | `http://localhost:3000` |
| `STORAGE_TYPE` | Тип хранилища (`local` или `s3`) | `local` |
| `STORAGE_LOCAL_PATH` | Путь для локальных файлов | `./uploads` |
| `STORAGE_LOCAL_URL` | Base URL для файлов | `http://localhost:8080/api/files` |

## Лимиты

| Параметр | Значение |
|---------|----------|
| Максимальный размер сообщения | 10 KB |
| WebSocket соединений на пользователя | 5 |
| Сообщений в секунду (WebSocket) | 10 |
| Изображения | 100 MB |
| Видео | 500 MB |
| Документы | 50 MB |
| Файлов на сообщение | 10 |

Константы определены в `internal/handlers/constants.go`.

## Модели данных

### Основные связи

```
User ←→ Chat (many-to-many через User1ID/User2ID)
Chat → Messages (one-to-many)
Message → Attachments (one-to-many)
Message → ReplyTo (self-referential)
User ↔ Contacts (many-to-many)
User → UnreadMessages (one-to-many)
```

### Soft Delete

Сообщения используют флаг `IsDeleted` вместо физического удаления для сохранения цепочек ответов.

## Roadmap

- [ ] Групповые чаты
- [ ] E2E шифрование
- [ ] Видео/аудио звонки (WebRTC)
- [ ] S3-совместимое хранилище для файлов
- [ ] Полнотекстовый поиск по сообщениям
- [ ] Push-уведомления
- [ ] Rate limiting на уровне пользователя
- [ ] Metrics (Prometheus)
- [ ] Distributed tracing

## Лицензия

MIT

## Контакты

GitHub Issues для bug reports и feature requests.
