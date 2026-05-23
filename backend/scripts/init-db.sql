-- Runs once on first container start.
-- pg_trgm powers fuzzy text search on patent titles / abstracts (Phase 1)
-- and trademark name matching (Phase 3).
CREATE EXTENSION IF NOT EXISTS pg_trgm;
