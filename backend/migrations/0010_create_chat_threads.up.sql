-- 0010: Chat threads — histórico persistido das conversas do assistente Argos.
--
-- Cada thread agrupa mensagens. Título é gerado a partir da primeira
-- mensagem do usuário (truncado em 60 chars) e pode ser editado.

CREATE TABLE IF NOT EXISTS chat_threads (
    id           BIGSERIAL    PRIMARY KEY,
    title        TEXT         NOT NULL,
    pinned       BOOLEAN      NOT NULL DEFAULT FALSE,
    archived     BOOLEAN      NOT NULL DEFAULT FALSE,
    message_count INT         NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_threads_updated ON chat_threads (updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_threads_archived ON chat_threads (archived);

CREATE TABLE IF NOT EXISTS chat_messages (
    id           BIGSERIAL    PRIMARY KEY,
    thread_id    BIGINT       NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
    role         TEXT         NOT NULL,                            -- 'user' | 'assistant' | 'system'
    content      TEXT         NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chat_messages_role_valid
        CHECK (role IN ('user', 'assistant', 'system'))
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_thread ON chat_messages (thread_id, created_at);

CREATE OR REPLACE FUNCTION chat_threads_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_chat_threads_updated_at ON chat_threads;
CREATE TRIGGER trg_chat_threads_updated_at
    BEFORE UPDATE ON chat_threads
    FOR EACH ROW EXECUTE FUNCTION chat_threads_touch_updated_at();
