-- 0002: Create the `trademarks` table.
-- Aligned with internal/domain/trademark.go.

CREATE TABLE IF NOT EXISTS trademarks (
    id               BIGSERIAL    PRIMARY KEY,
    process_number   TEXT         NOT NULL UNIQUE,
    name             TEXT         NOT NULL,
    normalized_name  TEXT         NOT NULL DEFAULT '',
    kind             TEXT         NOT NULL,
    status           TEXT         NOT NULL DEFAULT 'filed',
    owner            TEXT         NOT NULL DEFAULT '',
    nice_classes     INT[]        NOT NULL DEFAULT '{}',
    image_url        TEXT         NOT NULL DEFAULT '',
    filing_date      DATE,
    publication_date DATE,
    granted_date     DATE,
    rpi_issue        TEXT         NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT trademarks_kind_valid
        CHECK (kind IN ('nominative','figurative','mixed','three_dimensional')),
    CONSTRAINT trademarks_status_valid
        CHECK (status IN ('filed','published','granted','denied','archived','expired'))
);

CREATE INDEX IF NOT EXISTS idx_trademarks_status         ON trademarks (status);
CREATE INDEX IF NOT EXISTS idx_trademarks_filing_date    ON trademarks (filing_date DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_trademarks_rpi_issue      ON trademarks (rpi_issue);
CREATE INDEX IF NOT EXISTS idx_trademarks_name_trgm
    ON trademarks USING gin (normalized_name gin_trgm_ops);

CREATE OR REPLACE FUNCTION trademarks_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_trademarks_updated_at ON trademarks;
CREATE TRIGGER trg_trademarks_updated_at
    BEFORE UPDATE ON trademarks
    FOR EACH ROW EXECUTE FUNCTION trademarks_touch_updated_at();
