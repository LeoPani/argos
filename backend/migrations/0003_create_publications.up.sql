-- 0003: Create the `publications` table.
-- Unified table for Lens.org, UFOP OAI-PMH, Web of Science.

CREATE TABLE IF NOT EXISTS publications (
    id             BIGSERIAL    PRIMARY KEY,
    source         TEXT         NOT NULL,
    external_id    TEXT         NOT NULL,
    doi            TEXT         NOT NULL DEFAULT '',
    title          TEXT         NOT NULL,
    abstract       TEXT         NOT NULL DEFAULT '',
    authors        TEXT[]       NOT NULL DEFAULT '{}',
    affiliations   TEXT[]       NOT NULL DEFAULT '{}',
    kind           TEXT         NOT NULL DEFAULT 'article',
    journal        TEXT         NOT NULL DEFAULT '',
    published_date DATE,
    citation_count INT          NOT NULL DEFAULT 0,
    keywords       TEXT[]       NOT NULL DEFAULT '{}',
    url            TEXT         NOT NULL DEFAULT '',
    ipc_category   SMALLINT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT publications_source_externalid_unique UNIQUE (source, external_id),
    CONSTRAINT publications_source_valid
        CHECK (source IN ('lens','web_of_science','scielo','manual','ufop_oai')),
    CONSTRAINT publications_kind_valid
        CHECK (kind IN ('article','review','conference','book','thesis','preprint','other'))
);

CREATE INDEX IF NOT EXISTS idx_publications_source       ON publications (source);
CREATE INDEX IF NOT EXISTS idx_publications_published    ON publications (published_date DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_publications_ipc_category ON publications (ipc_category) WHERE ipc_category IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_publications_title_trgm
    ON publications USING gin (title gin_trgm_ops);

CREATE OR REPLACE FUNCTION publications_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_publications_updated_at ON publications;
CREATE TRIGGER trg_publications_updated_at
    BEFORE UPDATE ON publications
    FOR EACH ROW EXECUTE FUNCTION publications_touch_updated_at();
