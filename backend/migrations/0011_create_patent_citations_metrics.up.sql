-- 0011: Patent citations + computed metrics cache.
--
-- patent_citations armazena citações (forward = quem cita esta patente,
-- backward = quem esta patente cita), enriquecidas via Lens.org Patent API.
-- Schema espelha o que vem do Lens scholarly_citations[] e patent_citations[].

CREATE TABLE IF NOT EXISTS patent_citations (
    id              BIGSERIAL    PRIMARY KEY,
    source_patent_id BIGINT      REFERENCES patents(id) ON DELETE CASCADE,
    citation_kind   TEXT         NOT NULL,                          -- 'forward' | 'backward' | 'scientific'
    cited_lens_id   TEXT         NOT NULL DEFAULT '',               -- ID Lens da patente referenciada
    cited_app_number TEXT        NOT NULL DEFAULT '',               -- nº pedido (quando disponível)
    cited_title     TEXT         NOT NULL DEFAULT '',
    cited_year      INT,                                            -- ano de depósito da citada
    cited_ipc_codes TEXT[]       NOT NULL DEFAULT '{}',             -- p/ HJT originality/generality
    citation_date   DATE,                                           -- quando a citação foi feita
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT patent_citations_kind_valid
        CHECK (citation_kind IN ('forward', 'backward', 'scientific'))
);

CREATE INDEX IF NOT EXISTS idx_patent_citations_source ON patent_citations (source_patent_id);
CREATE INDEX IF NOT EXISTS idx_patent_citations_kind   ON patent_citations (citation_kind);
CREATE INDEX IF NOT EXISTS idx_patent_citations_year   ON patent_citations (cited_year);

-- patent_metrics cacheia indicadores composites por patente
-- (PCI Lanjouw-Schankerman 2004, HJT 2001, etc.) — recomputáveis on-demand.
CREATE TABLE IF NOT EXISTS patent_metrics (
    patent_id           BIGINT       PRIMARY KEY REFERENCES patents(id) ON DELETE CASCADE,
    forward_citations   INT          NOT NULL DEFAULT 0,
    backward_citations  INT          NOT NULL DEFAULT 0,
    scientific_citations INT         NOT NULL DEFAULT 0,
    family_size         INT          NOT NULL DEFAULT 0,            -- nº jurisdições
    claims_count        INT          NOT NULL DEFAULT 0,
    pci_score           NUMERIC(6,3),                               -- Lanjouw-Schankerman composite
    originality_index   NUMERIC(5,4),                               -- HJT 2001: 1 − Σ s²ⱼ (backward)
    generality_index    NUMERIC(5,4),                               -- HJT 2001: 1 − Σ s²ⱼ (forward)
    computed_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    source              TEXT         NOT NULL DEFAULT 'mock'        -- 'lens' | 'mock' | 'manual'
);

CREATE INDEX IF NOT EXISTS idx_patent_metrics_pci ON patent_metrics (pci_score DESC NULLS LAST);
