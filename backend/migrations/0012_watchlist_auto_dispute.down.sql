DROP TABLE IF EXISTS watchlist_auto_drafts;
ALTER TABLE watchlists DROP COLUMN IF EXISTS auto_dispute, DROP COLUMN IF EXISTS similarity_threshold;
