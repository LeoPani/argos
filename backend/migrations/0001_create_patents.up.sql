-- 0001: Create the core `patents` table.
-- Aligned with internal/domain/patent.go.

CREATE TABLE IF NOT EXISTS patents (
    id                 BIGSERIAL    PRIMARY KEY,
    application_number TEXT         NOT NULL UNIQUE,
    title              TEXT         NOT NULL,
    abstract           TEXT         NOT NULL,
    applicant          TEXT         NOT NULL DEFAULT '',
    inventors          TEXT[]       NOT NULL DEFAULT '{}',
    filing_date        DATE,
    publication_date   DATE,
    ipc_category       SMALLINT,                          -- 0..7 from AI, NULL if unclassified
    ipc_code           TEXT         NOT NULL DEFAULT '',  -- raw IPC code from INPI XML
    rpi_issue          TEXT         NOT NULL DEFAULT '',  -- e.g. "2750"
    status             TEXT         NOT NULL DEFAULT 'pending',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT patents_ipc_category_range
        CHECK (ipc_category IS NULL OR (ipc_category >= 0 AND ipc_category <= 7)),
    CONSTRAINT patents_status_valid
        CHECK (status IN ('pending', 'classified', 'failed', 'reclassified'))
);

-- Listing endpoints sort by publication date; this is the hottest index.
CREATE INDEX IF NOT EXISTS idx_patents_publication_date
    ON patents (publication_date DESC NULLS LAST);

-- Filtering by AI category.
CREATE INDEX IF NOT EXISTS idx_patents_ipc_category
    ON patents (ipc_category)
    WHERE ipc_category IS NOT NULL;

-- Filtering by RPI issue (dashboard "show me this week's patents").
CREATE INDEX IF NOT EXISTS idx_patents_rpi_issue
    ON patents (rpi_issue);

-- Status filter (e.g. "show me classification failures").
CREATE INDEX IF NOT EXISTS idx_patents_status
    ON patents (status);

-- Ensure pg_trgm is available — idempotent (no-op if já habilitado).
-- Necessário para trigram indexes abaixo.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram indexes for ILIKE search on title/abstract.
CREATE INDEX IF NOT EXISTS idx_patents_title_trgm
    ON patents USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_patents_abstract_trgm
    ON patents USING gin (abstract gin_trgm_ops);

-- Auto-bump updated_at on every UPDATE.
CREATE OR REPLACE FUNCTION patents_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_patents_touch_updated_at ON patents;
CREATE TRIGGER trg_patents_touch_updated_at
    BEFORE UPDATE ON patents
    FOR EACH ROW
    EXECUTE FUNCTION patents_touch_updated_at();
