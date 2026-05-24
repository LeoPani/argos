DROP TRIGGER IF EXISTS trg_chat_threads_updated_at ON chat_threads;
DROP FUNCTION IF EXISTS chat_threads_touch_updated_at;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_threads;
