DROP INDEX IF EXISTS idx_ufop_opp_classifier;
DROP INDEX IF EXISTS idx_ufop_opp_patentable;
ALTER TABLE ufop_opportunities
    DROP COLUMN IF EXISTS confidence,
    DROP COLUMN IF EXISTS classifier_version,
    DROP COLUMN IF EXISTS rationale,
    DROP COLUMN IF EXISTS is_patentable;
