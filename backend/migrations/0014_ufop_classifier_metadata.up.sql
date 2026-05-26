-- Migration 0014: rastreabilidade do classificador
--
-- Antes desta migration, o sistema mostrava "alta oportunidade IPC H" pra
-- trabalhos de Direito do Trabalho/INSS (falso positivo gritante). Razões:
--   1) keywords genéricas ("processo", "método", "sistema") batem em
--      qualquer texto de direito processual
--   2) heurística não verificava Art. 10 LPI (esquemas jurídicos não são
--      patenteáveis)
--   3) nenhum campo guardava QUAL classificador rodou (BERT vs Groq vs
--      heurística) nem POR QUE deu aquele resultado
--
-- Esta migration adiciona:
--   - is_patentable:        decisão binária (Art. 8 vs Art. 10 LPI)
--   - rationale:            justificativa textual (auditável, vem do LLM
--                           quando disponível, senão da heurística)
--   - classifier_version:   "bert-v1" | "groq-llama-3.3-70b" | "heuristic-v2"
--   - confidence:           0.0-1.0, do LLM ou inferida pela heurística

ALTER TABLE ufop_opportunities
    ADD COLUMN IF NOT EXISTS is_patentable      BOOLEAN,
    ADD COLUMN IF NOT EXISTS rationale          TEXT,
    ADD COLUMN IF NOT EXISTS classifier_version TEXT,
    ADD COLUMN IF NOT EXISTS confidence         FLOAT;

CREATE INDEX IF NOT EXISTS idx_ufop_opp_patentable
    ON ufop_opportunities (is_patentable) WHERE is_patentable IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ufop_opp_classifier
    ON ufop_opportunities (classifier_version);
