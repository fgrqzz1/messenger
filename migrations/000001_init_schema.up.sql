CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    login         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chats (
    id          BIGSERIAL PRIMARY KEY,
    type        TEXT NOT NULL CHECK (type IN ('direct', 'group')),
    title       TEXT,
    user_a_id   BIGINT REFERENCES users(id),
    user_b_id   BIGINT REFERENCES users(id),
    created_by  BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uniq_direct_chat
    ON chats (LEAST(user_a_id, user_b_id), GREATEST(user_a_id, user_b_id))
    WHERE type = 'direct';

CREATE TABLE chat_members (
    chat_id  BIGINT NOT NULL REFERENCES chats(id),
    user_id  BIGINT NOT NULL REFERENCES users(id),
    role     TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE messages (
    id             BIGSERIAL PRIMARY KEY,
    chat_id        BIGINT NOT NULL REFERENCES chats(id),
    sender_id      BIGINT NOT NULL REFERENCES users(id),
    client_msg_id  UUID NOT NULL,
    body           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chat_id, client_msg_id)
);

CREATE INDEX idx_messages_chat_id_created ON messages (chat_id, id DESC);

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_messages_body_trgm ON messages USING GIN (body gin_trgm_ops);
