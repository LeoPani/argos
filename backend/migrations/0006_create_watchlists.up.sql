-- 0006: Watchlists — monitoramento de termos, marcas, empresas e patentes.
--
-- Cada linha representa uma assinatura de alerta. O serviço varre as
-- bases (patents, trademarks) periodicamente comparando com last_check
-- e atualiza new_count + status.

CREATE TABLE IF NOT EXISTS watchlists (
    id           BIGSERIAL    PRIMARY KEY,
    label        TEXT         NOT NULL,                              -- string mostrada na UI
    watch_type   TEXT         NOT NULL,                              -- 'term' | 'brand' | 'company' | 'patent'
    query        TEXT         NOT NULL DEFAULT '',                   -- texto que será buscado (ILIKE)
    last_check   TIMESTAMPTZ,                                        -- null = nunca verificado
    new_count    INT          NOT NULL DEFAULT 0,                    -- novos itens desde last_check
    status       TEXT         NOT NULL DEFAULT 'ok',                 -- 'ok' | 'alert'

    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT watchlists_type_valid
        CHECK (watch_type IN ('term', 'brand', 'company', 'patent')),
    CONSTRAINT watchlists_status_valid
        CHECK (status IN ('ok', 'alert'))
);

CREATE INDEX IF NOT EXISTS idx_watchlists_status   ON watchlists (status);
CREATE INDEX IF NOT EXISTS idx_watchlists_type     ON watchlists (watch_type);
CREATE INDEX IF NOT EXISTS idx_watchlists_check    ON watchlists (last_check NULLS FIRST);

CREATE OR REPLACE FUNCTION watchlists_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_watchlists_updated_at ON watchlists;
CREATE TRIGGER trg_watchlists_updated_at
    BEFORE UPDATE ON watchlists
    FOR EACH ROW EXECUTE FUNCTION watchlists_touch_updated_at();
