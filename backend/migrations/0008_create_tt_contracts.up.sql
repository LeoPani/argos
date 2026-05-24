-- 0008: TT (Technology Transfer) contracts.
--
-- Modela contratos de transferência tecnológica entre a UFOP (titular)
-- e empresas licenciadas. Estrutura inspirada em contratos reais
-- celebrados por NITs brasileiros sob a Lei 10.973/2004 (Marco Legal).

CREATE TABLE IF NOT EXISTS tt_contracts (
    id                   BIGSERIAL    PRIMARY KEY,
    contract_number      TEXT         NOT NULL,                       -- ex: 'TT-UFOP-2026-001'
    patent_id            BIGINT       REFERENCES patents(id) ON DELETE SET NULL,
    pool_id              BIGINT,                                       -- FK adicionada após 0009

    -- Partes
    licensor             TEXT         NOT NULL DEFAULT 'Universidade Federal de Ouro Preto',
    licensee             TEXT         NOT NULL,
    licensee_cnpj        TEXT         NOT NULL DEFAULT '',

    -- Termos comerciais
    license_kind         TEXT         NOT NULL,                        -- 'exclusive' | 'non_exclusive' | 'sole'
    sublicensable        BOOLEAN      NOT NULL DEFAULT FALSE,
    territory            TEXT         NOT NULL DEFAULT 'BR',           -- 'BR' | 'LATAM' | 'WORLD' | etc
    field_of_use         TEXT         NOT NULL DEFAULT '',             -- restrição de campo (livre se vazio)
    royalty_rate         NUMERIC(5,2) NOT NULL DEFAULT 0,              -- % do faturamento líquido
    royalty_floor_annual NUMERIC(12,2) NOT NULL DEFAULT 0,             -- piso anual em BRL
    upfront_fee          NUMERIC(12,2) NOT NULL DEFAULT 0,             -- pagamento inicial em BRL

    -- Divisão UFOP × inventores (Lei 10.973: até 1/3 ao inventor)
    inventor_share_pct   INT          NOT NULL DEFAULT 33,             -- 0..50

    -- Marcos (JSONB array de {label, due_date, fee_brl, done})
    milestones           JSONB        NOT NULL DEFAULT '[]'::jsonb,

    -- Vigência
    signed_at            DATE,
    expires_at           DATE,

    -- Status
    status               TEXT         NOT NULL DEFAULT 'draft',
                                      -- 'draft' | 'negotiating' | 'active' | 'expired' | 'terminated'
    nit_approved         BOOLEAN      NOT NULL DEFAULT FALSE,          -- aprovação do NIT-UFOP
    audit_rights         BOOLEAN      NOT NULL DEFAULT TRUE,

    -- Anotações livres
    notes                TEXT         NOT NULL DEFAULT '',

    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT tt_contracts_number_unique UNIQUE (contract_number),
    CONSTRAINT tt_contracts_kind_valid
        CHECK (license_kind IN ('exclusive', 'non_exclusive', 'sole')),
    CONSTRAINT tt_contracts_status_valid
        CHECK (status IN ('draft', 'negotiating', 'active', 'expired', 'terminated')),
    CONSTRAINT tt_contracts_inventor_share_range
        CHECK (inventor_share_pct BETWEEN 0 AND 50),
    CONSTRAINT tt_contracts_target_present
        CHECK (patent_id IS NOT NULL OR pool_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_tt_contracts_patent ON tt_contracts (patent_id);
CREATE INDEX IF NOT EXISTS idx_tt_contracts_pool   ON tt_contracts (pool_id);
CREATE INDEX IF NOT EXISTS idx_tt_contracts_status ON tt_contracts (status);

CREATE OR REPLACE FUNCTION tt_contracts_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tt_contracts_updated_at ON tt_contracts;
CREATE TRIGGER trg_tt_contracts_updated_at
    BEFORE UPDATE ON tt_contracts
    FOR EACH ROW EXECUTE FUNCTION tt_contracts_touch_updated_at();
