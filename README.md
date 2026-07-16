# Messenger

Учебный pet-проект мессенджера: регистрация и вход, личные и групповые чаты, доставка сообщений в реальном времени по WebSocket, история с курсорной пагинацией, поиск по подстроке в рамках чата, статусы прочтения, реакции, редактирование сообщений, presence/last seen, светлая/тёмная тема.

Бэкенд — Go (Clean Architecture, `net/http`, PostgreSQL через `pgx`, `gorilla/websocket`). Фронтенд — React + TypeScript + CSS-модули (Vite), без UI-фреймворков.

Подробные материалы:

- [`docs/APP_OVERVIEW.md`](docs/APP_OVERVIEW.md) — пользовательские и технические сценарии
- [`docs/DATABASE.md`](docs/DATABASE.md) — схема БД, индексы, ключевые запросы
- [`docs/PROJECT_SUMMARY.md`](docs/PROJECT_SUMMARY.md) — архитектура, статус MVP, ограничения
- [`docs/ER_DIAGRAM.md`](docs/ER_DIAGRAM.md) — ER-диаграмма БД
- [`docs/AUTH_AND_ROLES.md`](docs/AUTH_AND_ROLES.md) — модель авторизации, роли и права доступа
- [`docs/REPORT.md`](docs/REPORT.md) — отчёт по практике
- [`web/README.md`](web/README.md) — локальная разработка фронтенда без Docker

## Стек

| Слой | Технологии |
|---|---|
| Backend | Go, `net/http` (ServeMux, Go 1.22+), `pgx`, `gorilla/websocket`, JWT (access/refresh), Argon2id |
| Frontend | React, TypeScript, Vite, CSS-модули |
| БД | PostgreSQL 16 (`pg_trgm` для поиска по подстроке) |
| Миграции | `golang-migrate` |
| Инфраструктура | Docker Compose (postgres, migrate, backend, frontend/nginx) |

## Архитектура бэкенда

Clean Architecture, зависимости только внутрь: **handler → service → repository**.

| Слой | Путь | Роль |
|---|---|---|
| Composition root | `cmd/server` | DI вручную, HTTP-сервер, graceful shutdown |
| Domain | `internal/domain` | модели, доменные ошибки, интерфейсы |
| Repository | `internal/repository/postgres` | SQL на pgx, маппинг в доменные ошибки |
| Service | `internal/service` | auth, chats, messages, members, search, read, profile |
| HTTP | `internal/handler/http` | REST-обработчики |
| WebSocket | `internal/handler/ws` | upgrade, auth первым фреймом, hub, broadcast |
| Пакеты | `pkg/jwt`, `pkg/password` | JWT (access/refresh), Argon2id |

Фронтенд (`web/src`): `api/` (REST + WS клиент), `components/`, `screens/`, `hooks/`, `context/`, `styles/`, `types/`, `utils/`.

REST — регистрация/логин/refresh, профиль, чаты, история, участники, поиск, read-state, редактирование, реакции. WebSocket (`GET /ws`) — отправка сообщений и realtime-события (`new_message`, `read`, `chat_updated`, `user_updated`, `presence`, `message_edited`, `reaction_updated`).

## Запуск через Docker Compose (рекомендуется)

Требуется установленный и запущенный **Docker Desktop** (на Windows — с включённым WSL2-бэкендом).

```bash
git clone <repo-url>
cd messenger
cp .env.example .env      # при необходимости поправить секреты/порт
docker compose up --build
```

Поднимаются по очереди: `postgres` → `migrate` (применяет все миграции из `migrations/` и завершается) → `backend` (слушает `:8080` внутри сети) → `frontend` (nginx на порту `${FRONTEND_PORT:-80}`, проксирует `/api/*` и `/ws` на backend).

Открыть в браузере: **http://localhost** (или `http://localhost:$FRONTEND_PORT`, если порт переопределён).

Остановить и убрать контейнеры/тома:

```bash
docker compose down -v
```

### Переменные окружения (`.env`)

| Переменная | Назначение | По умолчанию |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Учётные данные Postgres | `messenger` / `messenger` / `messenger` |
| `DATABASE_URL` | Строка подключения backend → Postgres | `postgres://messenger:messenger@postgres:5432/messenger?sslmode=disable` |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | Секреты подписи JWT (сменить перед реальным использованием) | `change-me-*` |
| `HTTP_ADDR` | Адрес, на котором слушает backend | `:8080` |
| `ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL` | Время жизни токенов | `15m` / `168h` |
| `FRONTEND_PORT` | Порт хоста, проброшенный на nginx фронтенда | `80` |
| `WS_ALLOWED_ORIGINS` | Список разрешённых Origin для WS (пусто = разрешены все, только для dev/LAN) | пусто |

## Наполнение БД

Схема создаётся автоматически сервисом `migrate` при первом `docker compose up` (5 миграций в `migrations/`, см. [`docs/DATABASE.md`](docs/DATABASE.md)). Тестовые данные не сидируются — пользователи создаются через экран регистрации (`POST /register`), чаты и сообщения — через обычный сценарий использования UI.

Применить/пересоздать миграции вручную (например, для локальной разработки без Docker):

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

## Локальная разработка без полного Docker-стека

Postgres — в Docker, backend и frontend — нативно:

```bash
docker compose up -d postgres
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/server

cd web
npm install
npm run dev   # http://localhost:5173, Vite проксирует REST/WS на :8080
```

Подробности — в [`web/README.md`](web/README.md) (доступ из LAN, хранение токенов, реализованные экраны).

## Тесты

```bash
go test ./...
```

E2E-сценарий бэкенда — `test/e2e`; unit/integration-тесты — в `internal/service`, `internal/repository`, `internal/handler`.

## Известные ограничения

Осознанные компромиссы MVP (подробнее — [`docs/PROJECT_SUMMARY.md`](docs/PROJECT_SUMMARY.md) §5): один инстанс сервера (presence и realtime-fan-out только в памяти процесса, без брокера сообщений), токены только в оперативной памяти браузера (без persist), нет revocation/blacklist JWT, нет вложений и push-уведомлений, нет удаления сообщений.
