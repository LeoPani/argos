DROP TRIGGER IF EXISTS trg_disputes_updated_at ON disputes;
DROP FUNCTION IF EXISTS disputes_touch_updated_at;
DROP TABLE IF EXISTS dispute_events;
DROP TABLE IF EXISTS dispute_documents;
DROP TABLE IF EXISTS dispute_parties;
DROP TABLE IF EXISTS disputes;
