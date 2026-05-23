-- Reverses 0001_create_patents.up.sql

DROP TRIGGER IF EXISTS trg_patents_touch_updated_at ON patents;
DROP FUNCTION IF EXISTS patents_touch_updated_at();
DROP TABLE IF EXISTS patents;
