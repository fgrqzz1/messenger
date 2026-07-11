DROP INDEX IF EXISTS idx_messages_body_trgm;
DROP EXTENSION IF EXISTS pg_trgm;
DROP INDEX IF EXISTS idx_messages_chat_id_created;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chat_members;
DROP INDEX IF EXISTS uniq_direct_chat;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS users;
