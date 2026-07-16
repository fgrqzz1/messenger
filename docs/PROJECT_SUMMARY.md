# Сводка по проекту Messenger

См. также: [DATABASE.md](./DATABASE.md) (схема БД), [APP_OVERVIEW.md](./APP_OVERVIEW.md) (сценарии работы), [spec.md](./spec.md) (исходное ТЗ).

## 1. Общее описание

Учебный pet-проект мессенджера: регистрация и вход, личные и групповые чаты, доставка сообщений в реальном времени по WebSocket, история с курсорной пагинацией и поиск по подстроке в рамках чата. Бэкенд — Go (Clean Architecture, `net/http`, PostgreSQL через `pgx`, `gorilla/websocket`), фронтенд — React + TypeScript + CSS-модули (Vite), без UI-фреймворков. Цель — портфолио с явной архитектурой и осознанными компромиссами single-instance MVP, а не production-масштабирование.

## 2. Архитектура

### Слои бэкенда (Clean Architecture)

Зависимости только внутрь: **handler → service → repository**. Интерфейсы репозиториев и realtime-абстракции (`RealtimeNotifier`, `PresenceChecker`) объявлены в `internal/domain`; реализации — в `internal/repository/postgres` и `internal/handler/ws`. Service не знает про HTTP/WS/SQL; handler не содержит бизнес-логики.

| Слой | Путь | Роль |
|------|------|------|
| Composition root | `cmd/server` | DI вручную, HTTP-сервер, graceful shutdown |
| Domain | `internal/domain` | модели, доменные ошибки, интерфейсы |
| Repository | `internal/repository/postgres` | SQL на pgx, маппинг в доменные ошибки |
| Service | `internal/service` | auth, chats, messages, members, search, read, profile |
| HTTP | `internal/handler/http` | REST (`net/http` ServeMux, паттерны Go 1.22+) |
| WebSocket | `internal/handler/ws` | upgrade, auth первым фреймом, hub, broadcast |
| Пакеты | `pkg/jwt`, `pkg/password` | JWT (access/refresh), Argon2id |

### Структура фронтенда

```
web/src/
  api/          — REST-клиент, WS-клиент, auth (токены в памяти)
  components/   — Sidebar, ChatWindow, MembersPanel, SearchPanel, …
  screens/      — Login, Profile
  hooks/        — useChats, useMessages, useWebSocket, useReadState, …
  context/      — Auth, ActiveChat, Users, Sidebar
  styles/       — tokens.css (CSS-переменные, светлая/тёмная тема)
  types/        — доменные TypeScript-типы
  utils/        — clientMsgId, theme, reactions, deliveryStatus, …
```

### Взаимодействие REST + WebSocket

- **REST** — регистрация/логин/refresh, профиль, чаты, история, участники, поиск, read-state, редактирование сообщений, реакции.
- **WebSocket** (`GET /ws`) — отправка сообщений (`send_message` → `ack` + `new_message`), realtime-события `read`, `chat_updated`, `user_updated`, `presence`, `message_edited`, `reaction_updated`.
- В Docker-стеке nginx фронтенда проксирует `/api/*` → backend и `/ws` → backend (`frontend/nginx.conf`).

## 3. Соответствие требованиям MVP

Сверка с условиями ниже (исходное ТЗ: [spec.md](./spec.md) §4–§5). Статусы — по коду репозитория, без приукрашивания.

| Требование | Статус | Комментарий |
|------------|--------|-------------|
| **Регистрация/логин: JWT + refresh, авторизация** | Реализовано | `POST /register`, `POST /login` → `access_token` + `refresh_token`; `POST /refresh`; middleware Bearer на защищённых ручках; WS-auth первым фреймом с токеном. |
| **Список чатов: личный 1-на-1, групповой с названием** | Реализовано | `POST /chats` (direct/group), `GET /chats` с превью последнего сообщения; direct переиспользуется при конфликте уникального индекса. |
| **Сообщения: real-time через WebSocket, история с пагинацией «вверх»** | Реализовано | Отправка только по WS; история `GET /chats/{id}/messages?before_id=&limit=`; на клиенте подгрузка при скролле вверх. |
| **Участники: добавление/удаление в группу только администратором** | Реализовано | `POST/DELETE …/members` через `ensureGroupChatAdmin`; не-admin → 403; для direct → validation. |
| **Поиск: по подстроке в рамках чата** | Реализовано | `GET /chats/{id}/search?q=` — `ILIKE`, лимит 50; GIN/trgm-индекс на `messages.body`. |
| **Авторизация: login+password → JWT; остальные ручки по токену; хэш пароля** | Реализовано | Не отдельный `api_key`, а JWT Bearer (`Authorization: Bearer <access_token>`). Пароль хранится как **Argon2id** (`pkg/password`, формат `$argon2id$v=…$m=…,t=…,p=…$…`). Публичны только `/register`, `/login` (+ health). |
| **БД и брокер сообщений** | Реализовано (осознанный выбор) | **PostgreSQL 16** — единственный источник истины (как в ТЗ). **Брокера сообщений нет**: realtime — in-memory WS-hub на одном инстансе. Для single-instance MVP это осознанное решение (spec §1, §9), а не упущение: нет горизонтального масштабирования и cross-instance fan-out. |

## 4. Что реализовано сверх обязательного минимума

По факту кода (не по желаемому списку):

- **Статусы прочтения** — таблица `chat_read_state`, `POST /chats/{id}/read`, `GET /chats/{id}/read-state`, WS `read`; unread в сайдбаре; на своих сообщениях UI ◌ → ✓ → ✓✓ («прочитано всеми» считается на клиенте по курсорам).
- **Presence / last_seen** — колонка `users.last_seen_at`, in-memory online в hub, WS `presence`, поля `online` / `last_seen_at` в `GET /chats/{id}/members`.
- **Редактирование сообщений** — `edited_at`, `PATCH /chats/{id}/messages/{message_id}`, WS `message_edited` (в исходном §1 spec — вне скоупа; добавлено позже миграцией и API).
- **Реакции** — `message_reactions` (`like` / `dislike` / `heart`), toggle REST + WS `reaction_updated`, агрегаты в истории.
- **Аватары** — клиентские, без загрузки файлов: цвет/инициалы от `userId` + login (`Avatar`).
- **Светлая / тёмная тема** — CSS `[data-theme]`, переключатель в профиле; предпочтение в `localStorage` (не токены).
- **Профиль** — `GET/PATCH /me`, `PATCH /me/password`; смена login с WS `user_updated`.
- **Поиск пользователей** — `GET /users/search` для создания direct-чата.
- **Смена названия группы** — `PATCH /chats/{id}` + WS `chat_updated`.
- **Docker Compose на весь стек** — `postgres` + `migrate` + `backend` + `frontend` (`docker-compose.yml`).
- **Индикатор WS** в сайдбаре, очередь неотправленных сообщений с `client_msg_id` при реконнекте.
- **Базовый responsive** (сайдбар-оверлей ≤768px).
- **E2E-сценарий** на бэкенде (`test/e2e`), unit/integration-тесты service/repository/handlers.
- **Health-check** `GET /health`, graceful shutdown HTTP + hub.

## 5. Известные ограничения и осознанные компромиссы

### Из исходного «вне скоупа» ([spec.md](./spec.md) §1, §9)

- Горизонтальное масштабирование, брокер сообщений, push-уведомления.
- **Revocation / blacklist токенов** — JWT stateless; смена пароля **не** инвалидирует уже выданные токены.
- Вложения / медиа, E2E-шифрование, полнотекстовый поиск через `tsvector`.
- **Удаление сообщений** — в коде не найдено (в отличие от редактирования, которое позже добавили).

### Зафиксированные в разработке / коде

- **Один инстанс сервера** — presence и fan-out только в памяти процесса.
- **WS-push best-effort** — оффлайн-получатели не получают persistent-очередь; источник истины — БД + REST-история.
- **Автодозагрузка истории при реконнекте при уже открытом чате** — на клиенте явно не найдена (история грузится при открытии чата / скролле вверх). Поведение «дыры» в открытой ленте после оффлайна — **требует проверки** вручную.
- **Нет WS-события «вас добавили в группу»** — чат может не появиться в сайдбаре до первого `new_message`, ручного обновления или релогина (описано в `web/README.md`).
- **Токены только в памяти** браузера — перезагрузка вкладки требует повторного входа.
- Корневой `README.md` почти пуст; подробности запуска — ниже и в `web/README.md` (список «не реализовано» в `web/README.md` **устарел** относительно текущего кода: read receipts, edit, theme уже есть).

## 6. Как запустить проект

Кратко (полный стек через Docker Compose):

1. Скопировать `.env.example` → `.env`, при необходимости поправить секреты.
2. Из корня репозитория: `docker compose up --build`.
3. UI: `http://localhost` (порт `FRONTEND_PORT`, по умолчанию 80). API за nginx: `/api/…`, WS: `/ws`.

Локальная разработка API/фронта без полного compose описана в [`web/README.md`](../web/README.md) (Postgres + `go run ./cmd/server` + `npm run dev` в `web/`). Миграции — образ `migrate/migrate` в compose или `golang-migrate` CLI.

Детали схемы БД — [DATABASE.md](./DATABASE.md). Пользовательские и технические сценарии — [APP_OVERVIEW.md](./APP_OVERVIEW.md).
