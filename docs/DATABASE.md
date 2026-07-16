# Описание базы данных

Схема собрана по **всем** миграциям в `migrations/` (не только по исходной §6 [spec.md](./spec.md)). Сводка по проекту: [PROJECT_SUMMARY.md](./PROJECT_SUMMARY.md). Сценарии: [APP_OVERVIEW.md](./APP_OVERVIEW.md).

Миграции применяются через `golang-migrate` (сервис `migrate` в `docker-compose.yml`).

| Файл | Содержание |
|------|------------|
| `000001_init_schema` | `users`, `chats`, `chat_members`, `messages`, индексы, `pg_trgm` |
| `000002_chat_read_state` | `chat_read_state` |
| `000003_user_last_seen` | `users.last_seen_at` |
| `000004_message_edited_at` | `messages.edited_at` |
| `000005_message_reactions` | `message_reactions` |

СУБД: **PostgreSQL** (в compose — образ `postgres:16-alpine`).

---

## Таблица `users`

**Назначение:** учётные записи пользователей.

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `id` | `BIGSERIAL` | `PRIMARY KEY` |
| `login` | `TEXT` | `UNIQUE NOT NULL` |
| `password_hash` | `TEXT` | `NOT NULL` (Argon2id-строка) |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |
| `last_seen_at` | `TIMESTAMPTZ` | nullable (миграция 000003) |

- **PK:** `id`
- **FK:** нет
- **Индексы:** уникальность `login` (через `UNIQUE`)
- **Связи:** один пользователь — много членств (`chat_members`), много сообщений (`messages.sender_id`), много реакций, много курсоров прочтения; в direct-чатах фигурирует как `user_a_id` / `user_b_id` / `created_by`

---

## Таблица `chats`

**Назначение:** чаты двух типов — личный (`direct`) и групповой (`group`).

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `id` | `BIGSERIAL` | `PRIMARY KEY` |
| `type` | `TEXT` | `NOT NULL`, `CHECK (type IN ('direct', 'group'))` |
| `title` | `TEXT` | nullable (обычно для group) |
| `user_a_id` | `BIGINT` | FK → `users(id)`, nullable |
| `user_b_id` | `BIGINT` | FK → `users(id)`, nullable |
| `created_by` | `BIGINT` | FK → `users(id)`, nullable (создатель группы) |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |

- **PK:** `id`
- **FK:** `user_a_id`, `user_b_id`, `created_by` → `users(id)`
- **Индексы:**
  - `uniq_direct_chat` — **UNIQUE** на `(LEAST(user_a_id, user_b_id), GREATEST(user_a_id, user_b_id))` **WHERE** `type = 'direct'` — один direct-чат на пару пользователей независимо от порядка id
- **Связи:** 1 чат → много участников (`chat_members`), много сообщений, много строк `chat_read_state`; для direct — ссылки на двух пользователей

---

## Таблица `chat_members`

**Назначение:** состав участников чата и роль (admin/member).

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `chat_id` | `BIGINT` | `NOT NULL`, FK → `chats(id)` |
| `user_id` | `BIGINT` | `NOT NULL`, FK → `users(id)` |
| `role` | `TEXT` | `NOT NULL DEFAULT 'member'`, `CHECK (role IN ('member', 'admin'))` |
| `joined_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |

- **PK:** составной `(chat_id, user_id)`
- **FK:** `chat_id` → `chats`; `user_id` → `users`
- **Индексы:** покрыт PK; отдельных вторичных индексов в миграциях нет
- **Связи:** **многие-ко-многим** между `users` и `chats` (через эту таблицу)

---

## Таблица `messages`

**Назначение:** сообщения в чатах; идемпотентность через `client_msg_id`.

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `id` | `BIGSERIAL` | `PRIMARY KEY` (монотонный порядок в чате) |
| `chat_id` | `BIGINT` | `NOT NULL`, FK → `chats(id)` |
| `sender_id` | `BIGINT` | `NOT NULL`, FK → `users(id)` |
| `client_msg_id` | `UUID` | `NOT NULL` |
| `body` | `TEXT` | `NOT NULL` |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |
| `edited_at` | `TIMESTAMPTZ` | nullable (миграция 000004; `NULL` = не редактировалось) |

- **PK:** `id`
- **FK:** `chat_id` → `chats`; `sender_id` → `users`
- **Уникальность:** `UNIQUE (chat_id, client_msg_id)` — идемпотентная повторная отправка
- **Индексы:**
  - `idx_messages_chat_id_created` — `(chat_id, id DESC)` — пагинация истории и «последнее сообщение»
  - `idx_messages_body_trgm` — **GIN** `(body gin_trgm_ops)` после `CREATE EXTENSION pg_trgm` — ускорение `ILIKE '%…%'` поиска
- **Связи:** 1 чат → много сообщений; 1 пользователь (sender) → много сообщений; 1 сообщение → много реакций

---

## Таблица `chat_read_state`

**Назначение:** курсор прочтения пользователя в чате (`last_read_message_id`).

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `chat_id` | `BIGINT` | `NOT NULL`, FK → `chats(id)` |
| `user_id` | `BIGINT` | `NOT NULL`, FK → `users(id)` |
| `last_read_message_id` | `BIGINT` | `NOT NULL DEFAULT 0` (логический курсор; не FK на `messages`) |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |

- **PK:** `(chat_id, user_id)`
- **FK:** `chat_id` → `chats`; `user_id` → `users`
- **Индексы:** PK
- **Связи:** по одной строке на пару (чат, пользователь); upsert с `GREATEST`, курсор назад не двигается
- **Замечание:** формальной FK на `messages(id)` нет — курсор может быть `0` или отставать/опережать по смыслу приложения

---

## Таблица `message_reactions`

**Назначение:** одна реакция пользователя на сообщение (toggle/replace).

| Колонка | Тип | Ограничения |
|---------|-----|-------------|
| `message_id` | `BIGINT` | `NOT NULL`, FK → `messages(id)` |
| `user_id` | `BIGINT` | `NOT NULL`, FK → `users(id)` |
| `reaction` | `TEXT` | `NOT NULL`, `CHECK (reaction IN ('like', 'dislike', 'heart'))` |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |

- **PK:** `(message_id, user_id)` — не больше одной реакции на пару (сообщение, пользователь)
- **FK:** `message_id` → `messages`; `user_id` → `users`
- **Индексы:** PK; отдельных индексов в миграции 000005 нет
- **Связи:** **многие-ко-многим** пользователи ↔ сообщения (с атрибутом `reaction`)

---

## Сводная таблица связей (для ER-диаграммы)

| Таблица A | Таблица B | Тип связи | Через поле / сущность |
|----------|-----------|-----------|------------------------|
| `users` | `chats` | 1 → много (опционально) | `chats.user_a_id` |
| `users` | `chats` | 1 → много (опционально) | `chats.user_b_id` |
| `users` | `chats` | 1 → много (опционально) | `chats.created_by` |
| `users` | `chats` | многие ↔ многие | `chat_members` (`user_id`, `chat_id`) |
| `chats` | `messages` | 1 → много | `messages.chat_id` |
| `users` | `messages` | 1 → много | `messages.sender_id` |
| `chats` | `chat_read_state` | 1 → много | `chat_read_state.chat_id` |
| `users` | `chat_read_state` | 1 → много | `chat_read_state.user_id` |
| `messages` | `message_reactions` | 1 → много | `message_reactions.message_id` |
| `users` | `message_reactions` | 1 → много | `message_reactions.user_id` |
| `messages` | `users` | многие ↔ многие (реакции) | `message_reactions` |

Смысловые кардинальности:

- Пользователь ↔ чат: **M:N** через `chat_members`
- Чат → сообщения: **1:N**
- Сообщение → реакции: **1:N** (уникально по пользователю)
- Чат/пользователь → read state: **1:1** на пару (составной PK)

---

## Ключевые запросы

### 1. Список чатов (LATERAL + read cursor)

Используется в `ChatRepository.ListByUser`:

```sql
SELECT c.id, c.type, c.title, lm.id, lm.body, lm.created_at,
       COALESCE(crs.last_read_message_id, 0)
FROM chats c
JOIN chat_members cm ON cm.chat_id = c.id AND cm.user_id = $1
LEFT JOIN LATERAL (
    SELECT id, body, created_at
    FROM messages m
    WHERE m.chat_id = c.id
    ORDER BY m.id DESC
    LIMIT 1
) lm ON true
LEFT JOIN chat_read_state crs ON crs.chat_id = c.id AND crs.user_id = $1
ORDER BY lm.created_at DESC NULLS LAST;
```

**Зачем так:** на каждый чат пользователя — ровно одно последнее сообщение без GROUP BY по всем сообщениям; `my_last_read_message_id` для unread на клиенте без отдельного запроса на чат. Сортировка по времени последнего сообщения.

### 2. Курсорная пагинация истории «вверх»

```sql
SELECT id, sender_id, body, created_at[, edited_at]
FROM messages
WHERE chat_id = $1 AND id < $2
ORDER BY id DESC
LIMIT $3;
```

(при первой странице — без `id < $2`, только `ORDER BY id DESC LIMIT`).

**Зачем так:** стабильный курсор по `BIGSERIAL id` (не по `created_at`); подгрузка более старых сообщений; индекс `(chat_id, id DESC)` обслуживает запрос.

### 3. Поиск по подстроке (ILIKE)

```sql
SELECT id, sender_id, body, created_at[, edited_at]
FROM messages
WHERE chat_id = $1 AND body ILIKE '%' || $2 || '%'
ORDER BY id DESC
LIMIT 50;
```

**Зачем так:** простой substring-search в рамках чата (ТЗ); GIN/trgm снижает стоимость `%query%`; лимит 50 ограничивает ответ. Не tsvector/FTS — осознанно вне исходного скоупа.

### 4. Идемпотентная вставка сообщения

```sql
INSERT INTO messages (chat_id, sender_id, client_msg_id, body)
VALUES ($1, $2, $3::uuid, $4)
ON CONFLICT (chat_id, client_msg_id) DO UPDATE
    SET body = messages.body
RETURNING …;
```

**Зачем так:** повторная отправка того же `client_msg_id` (реконнект, retry) возвращает уже сохранённую строку **без дубликата**; `DO UPDATE … SET body = messages.body` — no-op touch для получения `RETURNING` в PostgreSQL.

### 5. Upsert курсора прочтения (GREATEST)

```sql
INSERT INTO chat_read_state (chat_id, user_id, last_read_message_id)
VALUES ($1, $2, $3)
ON CONFLICT (chat_id, user_id) DO UPDATE SET
    last_read_message_id = GREATEST(
        chat_read_state.last_read_message_id,
        EXCLUDED.last_read_message_id
    ),
    updated_at = now()
RETURNING last_read_message_id;
```

**Зачем так:** курсор только вперёд; гонки/устаревшие клиентские значения не откатывают прочтение.

### 6. Агрегация реакций

```sql
SELECT
    message_id,
    COUNT(*) FILTER (WHERE reaction = 'like')::int,
    COUNT(*) FILTER (WHERE reaction = 'dislike')::int,
    COUNT(*) FILTER (WHERE reaction = 'heart')::int,
    MAX(reaction) FILTER (WHERE user_id = $viewer)
FROM message_reactions
WHERE message_id = ANY($1)
GROUP BY message_id;
```

**Зачем так:** один запрос на страницу истории вместо N; отдаёт счётчики и `my_reaction` для текущего зрителя. Toggle реакции — транзакция SELECT FOR UPDATE + INSERT/UPDATE/DELETE + тот же агрегат.

### 7. Уникальность direct-чата

Частичный unique-индекс на упорядоченную пару `(LEAST, GREATEST)` пользователей — создание второго direct между теми же людьми даёт conflict → сервис возвращает существующий чат.
