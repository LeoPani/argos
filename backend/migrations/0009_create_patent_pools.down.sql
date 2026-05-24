ALTER TABLE tt_contracts DROP CONSTRAINT IF EXISTS tt_contracts_pool_id_fkey;
DROP TRIGGER IF EXISTS trg_patent_pools_updated_at ON patent_pools;
DROP FUNCTION IF EXISTS patent_pools_touch_updated_at;
DROP TABLE IF EXISTS pool_members;
DROP TABLE IF EXISTS patent_pools;
