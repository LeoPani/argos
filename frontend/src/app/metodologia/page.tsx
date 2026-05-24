"use client";

// /metodologia — fonte única de verdade acadêmica.
// Lista cada métrica com fórmula, referências (papers seminais),
// requisitos de dados e interpretação. Defensável em banca.

import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useMethodology } from "@/lib/hooks";
import { BookOpen, ExternalLink, ArrowLeft, GraduationCap } from "lucide-react";

export default function MethodologyPage() {
  const { data, isLoading } = useMethodology();

  return (
    <div className="p-8 max-w-5xl space-y-6">
      <div className="flex items-center gap-2 text-sm">
        <Link href="/metricas" className="hover:text-white transition-colors"
          style={{ color: "var(--text-muted)" }}>
          <ArrowLeft size={14} className="inline mr-1" />
          Métricas
        </Link>
        <span style={{ color: "var(--text-muted)" }}>/</span>
        <span className="text-white">Metodologia</span>
      </div>

      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <GraduationCap size={24} />
          Metodologia &amp; Bibliografia
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Cada indicador segue uma metodologia validada por peer-review.
          Esta página documenta as fórmulas, papers de referência e requisitos de dados.
        </p>
      </div>

      {/* Disclaimer científico */}
      <Card style={{ borderColor: "#a855f730" }}>
        <div className="flex items-start gap-3">
          <BookOpen size={16} className="text-purple-400 mt-0.5 shrink-0" />
          <div className="text-sm" style={{ color: "var(--text-muted)" }}>
            <p className="text-white font-medium mb-1">Disclaimer científico</p>
            <p>
              Este painel implementa adaptações <span className="text-white">simplificadas</span> de metodologias
              estabelecidas na literatura de inovação. Algumas métricas (PCI completo, originality/generality
              clássicos) requerem dados de citações que dependem de integração com bases como Lens.org. Quando
              esses dados estão ausentes, indicamos com <Badge variant="warning">modo parcial</Badge>.
            </p>
          </div>
        </div>
      </Card>

      {isLoading && (
        <p className="text-sm text-center py-8" style={{ color: "var(--text-muted)" }}>
          Carregando metodologia…
        </p>
      )}

      {/* Lista de métricas */}
      {data?.metrics.map(m => (
        <div key={m.id} id={m.id} className="scroll-mt-8">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{m.name}</CardTitle>
            <Badge variant="muted">{m.id}</Badge>
          </CardHeader>

          <p className="text-sm mb-3" style={{ color: "var(--text-muted)" }}>
            {m.description}
          </p>

          {/* Fórmula */}
          <div className="mb-3">
            <p className="text-xs font-medium text-white mb-1">Fórmula</p>
            <pre className="p-3 rounded text-sm font-mono overflow-x-auto"
              style={{ background: "var(--surface-2)", color: "#a5b4fc" }}>
              {m.formula}
            </pre>
          </div>

          {/* Componentes */}
          {m.components && m.components.length > 0 && (
            <div className="mb-3">
              <p className="text-xs font-medium text-white mb-1.5">Componentes</p>
              <div className="space-y-1">
                {m.components.map(c => (
                  <div key={c.key} className="flex items-baseline gap-2 text-xs">
                    <span className="font-mono text-indigo-400 shrink-0 w-32">{c.key}</span>
                    <span className="text-white font-medium shrink-0 w-44">{c.label}</span>
                    <span style={{ color: "var(--text-muted)" }}>{c.definition}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {m.interpretation && (
            <div className="mb-3">
              <p className="text-xs font-medium text-white mb-1">Interpretação</p>
              <p className="text-sm" style={{ color: "var(--text-muted)" }}>{m.interpretation}</p>
            </div>
          )}

          {m.normalization && (
            <div className="mb-3">
              <p className="text-xs font-medium text-white mb-1">Normalização</p>
              <p className="text-sm" style={{ color: "var(--text-muted)" }}>{m.normalization}</p>
            </div>
          )}

          {m.data_requirements && (
            <div className="mb-3 p-2 rounded text-xs"
              style={{ background: "#fbbf2415", border: "1px solid #fbbf2440", color: "#fbbf24" }}>
              <span className="font-medium">Requisitos de dados:</span> {m.data_requirements}
            </div>
          )}

          {/* Referências */}
          <div className="pt-3" style={{ borderTop: "1px solid var(--border)" }}>
            <p className="text-xs font-medium text-white mb-1">Referências</p>
            <ol className="text-sm space-y-1.5 pl-4">
              {m.references.map((ref, i) => (
                <li key={i} className="list-decimal leading-relaxed"
                  style={{ color: "var(--text-muted)" }}>
                  {ref}
                </li>
              ))}
            </ol>
          </div>
        </Card>
        </div>
      ))}

      {/* Footer: como citar */}
      <Card style={{ borderColor: "var(--accent)" }}>
        <CardHeader>
          <CardTitle>Como citar este painel</CardTitle>
        </CardHeader>
        <pre className="text-xs p-3 rounded overflow-x-auto whitespace-pre-wrap"
          style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
{`Paniago, L. L. C. (${new Date().getFullYear()}).
Argos: Painel de inteligência de PI para a UFOP.
Indicadores AUTM/HJT/Triple Helix/Lanjouw-Schankerman
implementados via stack Go + Postgres + Next.js.
Repositório: github.com/LeoPani/argos`}
        </pre>
      </Card>

      <p className="text-xs text-center" style={{ color: "var(--text-muted)" }}>
        <a href="https://docs.api.lens.org/request-patent.html" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          Lens.org Patent API <ExternalLink size={9} />
        </a>
        {" · "}
        <a href="https://autm.net/surveys-and-tools/surveys/licensing-survey" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          AUTM Licensing Survey <ExternalLink size={9} />
        </a>
        {" · "}
        <a href="https://fortec-br.org" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          FORTEC <ExternalLink size={9} />
        </a>
      </p>
    </div>
  );
}
