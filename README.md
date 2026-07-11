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

Интеграционные тесты repository поднимают Postgres через testcontainers (нужен Docker):

```bash
go test ./...
```

### 4. Сборка

```bash
go build ./...
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
