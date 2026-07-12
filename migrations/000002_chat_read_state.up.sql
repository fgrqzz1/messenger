CREATE TABLE chat_read_state (
    chat_id              BIGINT NOT NULL REFERENCES chats(id),
    user_id              BIGINT NOT NULL REFERENCES users(id),
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);
