-- 0012: Watchlist auto-dispute mode.
--
-- When enabled, a check that finds matches with similarity above a threshold
-- automatically creates a *draft* dispute for human review. Standard practice
-- in commercial IP-watch services (e.g., CompuMark, Markify).

ALTER TABLE watchlists
    ADD COLUMN IF NOT EXISTS auto_dispute      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS similarity_threshold INT  NOT NULL DEFAULT 70;

-- Histórico de disputas geradas automaticamente (pra audit trail)
CREATE TABLE IF NOT EXISTS watchlist_auto_drafts (
    id           BIGSERIAL    PRIMARY KEY,
    watchlist_id BIGINT       NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    dispute_id   BIGINT       NOT NULL REFERENCES disputes(id)    ON DELETE CASCADE,
    similarity_pct INT        NOT NULL,
    matched_kind TEXT         NOT NULL,    -- 'patent' | 'trademark'
    matched_id   BIGINT       NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wl_auto_drafts_wl  ON watchlist_auto_drafts (watchlist_id);
CREATE INDEX IF NOT EXISTS idx_wl_auto_drafts_dsp ON watchlist_auto_drafts (dispute_id);
