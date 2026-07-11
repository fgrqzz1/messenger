# Messenger (MVP)

Учебный pet-проект: бэкенд мессенджера с личными и групповыми чатами и React-фронтендом.

## Стек

- Go 1.26
- PostgreSQL 16
- **pgx/v5** (`github.com/jackc/pgx/v5`) — драйвер и connection pool для repository-слоя
- **golang-migrate** — SQL-миграции (`migrations/`)
- React + TypeScript (Vite) — SPA в `web/`
- Docker Compose — полный стек одной командой

## Быстрый старт (Docker)
Единственная команда для полного запуска (Postgres + миграции + backend + frontend):

```bash
cp .env.example .env   # один раз; задайте свои секреты
docker compose up --build
```

Откройте в браузере: **http://localhost** (или порт из `FRONTEND_PORT` в `.env`).

Остановка и полная очистка (включая volume БД):

```bash
docker compose down -v
```

Повторный подъём после `down -v` идемпотентен — миграции применяются автоматически сервисом `migrate`.

### Порты

| Сервис | Наружу | Внутри docker-сети |
|---|---|---|
| `frontend` (nginx) | `FRONTEND_PORT` → 80 (по умолчанию **80**) | — |
| `backend` | не пробрасывается | `backend:8080` |
| `postgres` | не пробрасывается | `postgres:5432` |

Nginx проксирует `/api/*` → backend REST (без префикса `/api`, см. `docs/spec.md` §7) и `/ws` → WebSocket.

### Миграции вручную

Миграции накатываются сервисом `migrate` при `docker compose up`. Повторно:

```bash
docker compose run --rm migrate
```

Откат (осторожно):

```bash
docker compose run --rm migrate -path=/migrations -database="$DATABASE_URL" down 1
```

## Локальная разработка без полного стека

### Только Postgres

В `.env` для локального backend укажите `DATABASE_URL` с `localhost` (и при необходимости временно пробросьте порт Postgres).

```bash
docker compose up -d postgres
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/server
```

Фронтенд:

```bash
cd web && npm ci && npm run dev
```

Vite-прокси в `web/vite.config.ts` направляет REST и `/ws` на `localhost:8080`.

### Тесты

```bash
go test ./...
go test -v ./test/e2e/...
```

## Структура проекта

```
/cmd/server          — main.go, DI, graceful shutdown, GET /health
/internal            — domain / repository / service / handler
/pkg                 — password (Argon2id), jwt
/migrations          — SQL (golang-migrate)
/web                 — React SPA
/backend/Dockerfile  — multi-stage сборка Go-бинаря
/frontend            — Dockerfile + nginx.conf
/docker-compose.yml  — postgres, migrate, backend, frontend
```

## Формат ошибок API

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

## REST и WebSocket

| Ручка | Описание |
|---|---|
| `POST /register`, `POST /login`, `POST /refresh` | Auth |
| `GET/POST /chats`, сообщения, участники, поиск | См. `docs/spec.md` §7 |
| `GET /ws` | WebSocket: первый фрейм `{"token":"..."}`, затем `send_message` → `ack` / `new_message` |
| `GET /health` | Liveness/readiness: 200 если БД доступна (без auth) |

Через nginx с фронтенда: `/api/register` → `/register`, `/ws` без изменений.

### Синхронизация списка чатов (фронтенд)

- **Создание чата** (`POST /chats`): ответ сразу вставляется в локальный список сайдбара (`upsertCreatedChat`), без ожидания WS или повторного `GET /chats`.
- **Входящее `new_message`**: если `chat_id` уже есть в списке — обновляется превью; если чата ещё нет (собеседник только что создал direct или вас добавили в группу и пришло первое сообщение) — тихий рефетч `GET /chats`, чтобы чат появился в сайдбаре.
- **Добавление в группу без сообщений** (`POST /chats/{id}/members`): в ТЗ нет WS-события о членстве (только `ack` / `new_message`), поэтому добавленный пользователь **не увидит** чат, пока не придёт сообщение или не будет релогин/`GET /chats`. Это известное ограничение MVP; отдельный этап на бэкенде (например, push `member_added` / `chat_updated`).

## Принятые допущения

- Для первой страницы истории (`before_id <= 0`) — тот же `ORDER BY id DESC` без `id < $cursor`.
- В пагинации/поиске `chat_id` подставляется из аргумента метода.
- Секреты и креды БД — только из `.env` (`env_file` в Compose), не захардкожены в Dockerfile/compose.

## Вне скоупа MVP

См. `docs/spec.md` §1 и §9: горизонтальное масштабирование, read receipts, push, вложения, E2E, tsvector-поиск и др.
