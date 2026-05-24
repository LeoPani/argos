-- 0009: Patent pools — agregação de várias patentes UFOP em um único bundle.
--
-- Cada pool tem termos agregados (royalty, duração, tipo).
-- A tabela pool_members rateia a participação de cada patente no pool.

CREATE TABLE IF NOT EXISTS patent_pools (
    id                   BIGSERIAL    PRIMARY KEY,
    name                 TEXT         NOT NULL,                       -- ex: 'Pool Biotecnologia UFOP'
    description          TEXT         NOT NULL DEFAULT '',
    pool_kind            TEXT         NOT NULL DEFAULT 'offensive',   -- 'offensive' | 'defensive' | 'standard_essential'

    -- Termos agregados (sobreescrevíveis por contrato individual)
    royalty_rate         NUMERIC(5,2) NOT NULL DEFAULT 0,              -- % agregado
    territory            TEXT         NOT NULL DEFAULT 'BR',
    duration_years       INT          NOT NULL DEFAULT 10,

    administrator        TEXT         NOT NULL DEFAULT 'NIT-UFOP',
    status               TEXT         NOT NULL DEFAULT 'forming',     -- 'forming' | 'active' | 'closed'

    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT patent_pools_kind_valid
        CHECK (pool_kind IN ('offensive', 'defensive', 'standard_essential')),
    CONSTRAINT patent_pools_status_valid
        CHECK (status IN ('forming', 'active', 'closed'))
);

-- Membros do pool com rateio de royalty
CREATE TABLE IF NOT EXISTS pool_members (
    id            BIGSERIAL    PRIMARY KEY,
    pool_id       BIGINT       NOT NULL REFERENCES patent_pools(id) ON DELETE CASCADE,
    patent_id     BIGINT       NOT NULL REFERENCES patents(id)      ON DELETE CASCADE,
    share_pct     NUMERIC(5,2) NOT NULL DEFAULT 0,                   -- % do pool atribuído a esta patente
    added_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pool_members_unique UNIQUE (pool_id, patent_id),
    CONSTRAINT pool_members_share_range
        CHECK (share_pct BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_pool_members_pool   ON pool_members (pool_id);
CREATE INDEX IF NOT EXISTS idx_pool_members_patent ON pool_members (patent_id);
CREATE INDEX IF NOT EXISTS idx_patent_pools_status ON patent_pools (status);

CREATE OR REPLACE FUNCTION patent_pools_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_patent_pools_updated_at ON patent_pools;
CREATE TRIGGER trg_patent_pools_updated_at
    BEFORE UPDATE ON patent_pools
    FOR EACH ROW EXECUTE FUNCTION patent_pools_touch_updated_at();

-- Agora podemos completar a FK em tt_contracts.pool_id
ALTER TABLE tt_contracts
    DROP CONSTRAINT IF EXISTS tt_contracts_pool_id_fkey;
ALTER TABLE tt_contracts
    ADD CONSTRAINT tt_contracts_pool_id_fkey
    FOREIGN KEY (pool_id) REFERENCES patent_pools(id) ON DELETE SET NULL;
