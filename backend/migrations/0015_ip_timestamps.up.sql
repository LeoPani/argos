-- 0015: IP Timestamps — registro de anterioridade com cadeia de hashes
-- Funciona como prova de existência (Proof-of-Existence):
-- cada registro armazena SHA-256 do conteúdo + aponta para o hash anterior,
-- criando uma cadeia imutável auditável sem blockchain real.

CREATE TABLE IF NOT EXISTS ip_timestamps (
    id           BIGSERIAL PRIMARY KEY,
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    authors      TEXT[]      NOT NULL DEFAULT '{}',
    category     TEXT        NOT NULL DEFAULT 'invenção',  -- invenção | software | design | segredo industrial
    content_hash TEXT        NOT NULL UNIQUE,  -- SHA-256(title|description|authors|created_at)
    prev_hash    TEXT,                          -- hash do registro anterior (cadeia)
    chain_index  BIGINT      NOT NULL DEFAULT 0,  -- posição na cadeia
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ip_timestamps_created_at_idx ON ip_timestamps (created_at DESC);
CREATE INDEX IF NOT EXISTS ip_timestamps_chain_idx      ON ip_timestamps (chain_index);
