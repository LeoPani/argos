-- 0004: Arbitration / dispute tables.
-- Aligned with internal/domain/dispute.go.

CREATE TABLE IF NOT EXISTS disputes (
    id           BIGSERIAL    PRIMARY KEY,
    case_number  TEXT         NOT NULL UNIQUE,
    title        TEXT         NOT NULL,
    summary      TEXT         NOT NULL DEFAULT '',
    kind         TEXT         NOT NULL,
    status       TEXT         NOT NULL DEFAULT 'open',
    patent_id    BIGINT       REFERENCES patents(id)     ON DELETE SET NULL,
    trademark_id BIGINT       REFERENCES trademarks(id)  ON DELETE SET NULL,
    opened_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT disputes_kind_valid
        CHECK (kind IN ('trademark_infringement','patent_infringement','authorship','licensing','other')),
    CONSTRAINT disputes_status_valid
        CHECK (status IN ('open','in_review','awaiting_info','resolved','withdrawn','escalated'))
);

CREATE TABLE IF NOT EXISTS dispute_parties (
    id         BIGSERIAL    PRIMARY KEY,
    dispute_id BIGINT       NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    name       TEXT         NOT NULL,
    role       TEXT         NOT NULL,
    email      TEXT         NOT NULL DEFAULT '',
    document   TEXT         NOT NULL DEFAULT '',
    joined_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dispute_documents (
    id           BIGSERIAL    PRIMARY KEY,
    dispute_id   BIGINT       NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    uploaded_by  BIGINT,
    title        TEXT         NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    storage_path TEXT         NOT NULL DEFAULT '',
    hash_sha256  TEXT         NOT NULL DEFAULT '',
    size_bytes   BIGINT       NOT NULL DEFAULT 0,
    mime_type    TEXT         NOT NULL DEFAULT '',
    uploaded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dispute_events (
    id          BIGSERIAL    PRIMARY KEY,
    dispute_id  BIGINT       NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    actor_id    BIGINT,
    event_type  TEXT         NOT NULL,
    payload     TEXT         NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_disputes_status        ON disputes (status);
CREATE INDEX IF NOT EXISTS idx_disputes_kind          ON disputes (kind);
CREATE INDEX IF NOT EXISTS idx_disputes_opened_at     ON disputes (opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_dispute_parties_did    ON dispute_parties (dispute_id);
CREATE INDEX IF NOT EXISTS idx_dispute_documents_did  ON dispute_documents (dispute_id);
CREATE INDEX IF NOT EXISTS idx_dispute_events_did     ON dispute_events (dispute_id);

CREATE OR REPLACE FUNCTION disputes_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_disputes_updated_at ON disputes;
CREATE TRIGGER trg_disputes_updated_at
    BEFORE UPDATE ON disputes
    FOR EACH ROW EXECUTE FUNCTION disputes_touch_updated_at();
