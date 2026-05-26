-- Migration 0013: ingest das Revistas da Propriedade Industrial (RPI) do INPI
--
-- Cada RPI é publicada semanalmente em revistas.inpi.gov.br com despachos
-- de patentes e marcas. Esta tabela guarda registros extraídos das últimas
-- N revistas para construir um dataset real e auditável (vs scraping do
-- Google Patents que tem rate-limit).
--
-- Granularidade: 1 linha = 1 despacho (patente OU marca OU desenho ind).

CREATE TABLE IF NOT EXISTS inpi_publications (
    id              BIGSERIAL PRIMARY KEY,

    -- Metadata da RPI
    rpi_number      INTEGER NOT NULL,                -- ex: 2914
    rpi_date        DATE,                            -- data de publicação
    rpi_section     TEXT,                            -- "patentes" | "marcas" | "des_ind"

    -- Identificadores do despacho
    process_number  TEXT NOT NULL,                   -- ex: BR102019001234-5
    despacho_code   TEXT,                            -- ex: 6.6.1, 11.2 (Manual INPI)
    despacho_desc   TEXT,                            -- descrição textual do código

    -- Conteúdo extraído
    title           TEXT,                            -- título da invenção/marca
    applicant       TEXT,                            -- depositante / titular
    inventor        TEXT,                            -- inventor (patentes)
    ipc_codes       TEXT[],                          -- IPCs do despacho
    nice_class      INTEGER[],                       -- classes Nice (marcas)
    raw_text        TEXT,                            -- bloco original (auditoria)

    -- Flags
    is_ufop         BOOLEAN NOT NULL DEFAULT FALSE,  -- pré-filtrado por keyword UFOP

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (rpi_number, process_number, despacho_code)
);

CREATE INDEX IF NOT EXISTS idx_inpi_pub_rpi         ON inpi_publications (rpi_number);
CREATE INDEX IF NOT EXISTS idx_inpi_pub_process     ON inpi_publications (process_number);
CREATE INDEX IF NOT EXISTS idx_inpi_pub_is_ufop     ON inpi_publications (is_ufop) WHERE is_ufop;
CREATE INDEX IF NOT EXISTS idx_inpi_pub_applicant   ON inpi_publications USING gin (applicant gin_trgm_ops);
