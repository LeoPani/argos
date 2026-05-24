-- 0007: Arbitration subjects + AI verdicts
--
-- Each dispute may have N "subjects" (trademarks/patents/etc.) competing
-- for the same right. The AI analysis result is stored as a verdict.

CREATE TABLE IF NOT EXISTS dispute_subjects (
    id           BIGSERIAL    PRIMARY KEY,
    dispute_id   BIGINT       NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    kind         TEXT         NOT NULL,                              -- 'trademark' | 'patent' | 'inventor' | 'other'
    ref_id       BIGINT,                                             -- FK to trademarks.id / patents.id (null for free-text)
    label        TEXT         NOT NULL,                              -- display name (mirrored for resilience)
    party_id     BIGINT       REFERENCES dispute_parties(id) ON DELETE SET NULL,
    metadata     JSONB        NOT NULL DEFAULT '{}'::jsonb,          -- arbitrary attributes (filing date snapshot, etc.)
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT dispute_subjects_kind_valid
        CHECK (kind IN ('trademark', 'patent', 'inventor', 'other'))
);

CREATE INDEX IF NOT EXISTS idx_dispute_subjects_did   ON dispute_subjects (dispute_id);
CREATE INDEX IF NOT EXISTS idx_dispute_subjects_kind  ON dispute_subjects (kind);

-- AI verdicts (re-runnable; we keep history)
CREATE TABLE IF NOT EXISTS arbitration_verdicts (
    id              BIGSERIAL   PRIMARY KEY,
    dispute_id      BIGINT      NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    winner_subject_id BIGINT    REFERENCES dispute_subjects(id) ON DELETE SET NULL,
    confidence      INT         NOT NULL DEFAULT 0,                  -- 0..100
    method          TEXT        NOT NULL DEFAULT 'heuristic_v1',     -- 'heuristic_v1' | 'claude_v1' | 'hybrid'
    summary         TEXT        NOT NULL DEFAULT '',                 -- short human summary
    reasoning       JSONB       NOT NULL DEFAULT '{}'::jsonb,        -- structured per-subject breakdown
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_arbitration_verdicts_did ON arbitration_verdicts (dispute_id, created_at DESC);
