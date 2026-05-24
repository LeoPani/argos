-- 0005: UFOP Intelligence — oportunidades de PI detectadas por IA.
--
-- Cada linha representa uma publicação / notícia da UFOP que foi
-- analisada pelo pipeline e classificada com potencial de PI.

CREATE TABLE IF NOT EXISTS ufop_opportunities (
    id                BIGSERIAL    PRIMARY KEY,
    source            TEXT         NOT NULL,               -- 'oai' | 'portal' | 'lens'
    external_id       TEXT         NOT NULL,               -- OAI identifier ou URL
    title             TEXT         NOT NULL,
    authors           TEXT[]       NOT NULL DEFAULT '{}',
    department        TEXT         NOT NULL DEFAULT '',
    abstract          TEXT         NOT NULL DEFAULT '',
    url               TEXT         NOT NULL DEFAULT '',
    published_at      DATE,

    -- AI analysis
    ipc_suggestion    TEXT         NOT NULL DEFAULT '',    -- ex: "C22B / G06F"
    ipc_category      SMALLINT,                            -- BERT output (0..7)
    opportunity_level TEXT         NOT NULL DEFAULT 'low', -- 'high' | 'medium' | 'low'
    similarity_pct    INT          NOT NULL DEFAULT 0,     -- vs. existing patents (0-100)
    pi_score          FLOAT        NOT NULL DEFAULT 0,     -- composite 0-10
    ai_analysis       TEXT         NOT NULL DEFAULT '',    -- insight gerado

    -- Lifecycle
    status            TEXT         NOT NULL DEFAULT 'new', -- 'new' | 'reviewed' | 'converted' | 'dismissed'
    publication_id    BIGINT       REFERENCES publications(id) ON DELETE SET NULL,

    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT ufop_opp_source_external_unique UNIQUE (source, external_id),
    CONSTRAINT ufop_opp_level_valid
        CHECK (opportunity_level IN ('high', 'medium', 'low')),
    CONSTRAINT ufop_opp_status_valid
        CHECK (status IN ('new', 'reviewed', 'converted', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_ufop_opp_level     ON ufop_opportunities (opportunity_level);
CREATE INDEX IF NOT EXISTS idx_ufop_opp_status    ON ufop_opportunities (status);
CREATE INDEX IF NOT EXISTS idx_ufop_opp_published ON ufop_opportunities (published_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_ufop_opp_score     ON ufop_opportunities (pi_score DESC);
CREATE INDEX IF NOT EXISTS idx_ufop_opp_source    ON ufop_opportunities (source);

CREATE OR REPLACE FUNCTION ufop_opportunities_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ufop_opp_updated_at ON ufop_opportunities;
CREATE TRIGGER trg_ufop_opp_updated_at
    BEFORE UPDATE ON ufop_opportunities
    FOR EACH ROW EXECUTE FUNCTION ufop_opportunities_touch_updated_at();
