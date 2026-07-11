# Messenger (MVP)

Учебный pet-проект: бэкенд мессенджера с личными и групповыми чатами.

## Стек

- Go 1.26
- PostgreSQL 16
- **pgx/v5** (`github.com/jackc/pgx/v5`) — драйвер и connection pool для repository-слоя
- **golang-migrate** — SQL-миграции (`migrations/`)
- **testcontainers-go** — интеграционные тесты repository на реальной Postgres

## Структура проекта

```
/internal
  /domain            — модели, доменные ошибки, интерфейсы репозиториев
  /repository
    /postgres        — реализация репозиториев на pgx
  /service           — бизнес-логика (auth, chat, message, member, search)
  /handler
    /http            — REST-хендлеры, middleware JWT, маппинг ошибок
    /ws              — WebSocket-хендлер, hub для realtime-доставки
/cmd
  /server            — main.go, DI и graceful shutdown
/pkg
  /password          — Argon2id-хэширование паролей
  /jwt               — выпуск и проверка access/refresh JWT
/migrations          — SQL-миграции (golang-migrate)
/docker-compose.yml  — Postgres для локальной разработки
```

## Быстрый старт

### 1. Postgres

```bash
docker compose up -d
```

### 2. Миграции

Установите [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) и примените схему:

```bash
cp .env.example .env
migrate -path migrations -database "$DATABASE_URL" up
```

### 3. Тесты

Интеграционные тесты repository и сквозной e2e-сценарий поднимают Postgres через testcontainers (нужен Docker):

```bash
go test ./...
```

Сквозной сценарий (Этап 5) — `test/e2e/scenario_test.go`:

1. Регистрация и логин двух пользователей (access + refresh token)
2. Создание группового чата, добавление участника
3. WebSocket: аутентификация первым фреймом, `send_message` → `ack` + `new_message`
4. Идемпотентность по `client_msg_id` (проверка COUNT в таблице `messages`)
5. Пагинация истории (`before_id`) для обоих участников
6. Поиск по подстроке в чате
7. Удаление участника: 403 для не-admin, 204 для admin

```bash
go test -v ./test/e2e/...
```

### 4. Сборка и запуск

```bash
go build -o bin/server ./cmd/server
# задайте переменные из .env.example
./bin/server
```

```bash
go build ./...
```

## Формат ошибок API

Все REST-ручки при ошибке возвращают JSON единого вида:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Неверные данные"
  }
}
```

| HTTP | `error.code` | Когда |
|---|---|---|
| 400 | `validation_error` | Невалидный JSON/параметры |
| 401 | `unauthorized` | Нет или невалидный access/refresh token |
| 401 | `invalid_credentials` | Неверный login/password при `/login` |
| 403 | `forbidden` | Нет прав (не участник чата, не admin) |
| 404 | `not_found` | Сущность не найдена |
| 409 | `conflict` | Конфликт (дубликат login, участник уже в чате) |
| 500 | `internal_error` | Неожиданная ошибка сервера |

## Handler layer (Этап 4)

### REST (`internal/handler/http/`)

| Ручка | Хендлер |
|---|---|
| `POST /register` | `Register` |
| `POST /login` | `Login` |
| `POST /refresh` | `Refresh` (Bearer refresh token) |
| `GET /chats` | `ListChats` |
| `POST /chats` | `CreateChat` (`type`: `direct` \| `group`) |
| `GET /chats/{id}/messages` | `ListMessages` (`before_id`, `limit`) |
| `POST /chats/{id}/members` | `AddMember` |
| `DELETE /chats/{id}/members/{user_id}` | `RemoveMember` |
| `GET /chats/{id}/search` | `SearchMessages` (`q`) |

JWT-middleware (`Auth`) защищает все ручки, кроме `/register`, `/login`, `/refresh`.

### WebSocket (`internal/handler/ws/`)

- `GET /ws` — первый фрейм `{"token":"<access_token>"}`, таймаут аутентификации 5 с
- `send_message` → синхронный `ack` с `server_id` после записи в БД
- `new_message` — push онлайн-участникам чата через `Hub`
- Graceful shutdown закрывает hub и активные соединения

```bash
go test ./internal/handler/...
```

## Repository layer (Этап 2)

Реализованы интерфейсы из `internal/domain/repository.go`:

| Репозиторий | Особенности |
|---|---|
| `UserRepository` | CRUD по login/id, `ErrConflict` при дубликате login |
| `ChatRepository` | direct/group в транзакции с участниками, список через LATERAL join |
| `MessageRepository` | идемпотентный `Create` через `ON CONFLICT (chat_id, client_msg_id)`, курсорная пагинация, ILIKE-поиск |
| `MemberRepository` | добавление/удаление/получение участника |

SQL-запросы для списка чатов, пагинации и поиска соответствуют `docs/spec.md` §6.

Доменные ошибки (`ErrNotFound`, `ErrConflict`) возвращаются наружу; `pgx.ErrNoRows` и коды PostgreSQL не протекают из пакета.

## Service layer (Этап 3)

Бизнес-логика в `internal/service/`:

| Модуль | Методы |
|---|---|
| `auth.go` | `Register` (Argon2id), `Login`, `Refresh` (stateless JWT) |
| `chat.go` | `CreateDirectChat` (переиспользование через `GetDirectByUsers` при конфликте), `CreateGroupChat`, `GetChats` |
| `message.go` | `SendMessage` (идемпотентность по `client_msg_id` через repository), `GetMessageHistory` |
| `member.go` | `AddMember`, `RemoveMember` (только `role=admin`) |
| `search.go` | `Search` (только для участников чата) |

JWT TTL задаётся при создании `pkg/jwt.Manager` (по умолчанию в handler-этапе — из env: `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`).

### Тестирование service-слоя

Юнит-тесты в `internal/service/*_test.go` используют **ручные моки** интерфейсов из `internal/domain` (файл `mocks_test.go`). Генераторы mockery/gomock не применяются — зависимости подменяются тестовыми структурами с функциями-заглушками.

```bash
go test ./internal/service/...
```

## Принятые допущения

- Для первой страницы истории (`before_id <= 0`) используется тот же порядок `ORDER BY id DESC`, но без условия `id < $cursor` — в spec явно описан только курсорный запрос.
- В результатах пагинации/поиска по spec выбираются `id, sender_id, body, created_at`; `chat_id` подставляется из аргумента метода.

## Вне скоупа MVP

См. `docs/spec.md` §1 и §9: горизонтальное масштабирование, read receipts, push, вложения, E2E, tsvector-поиск и др.
