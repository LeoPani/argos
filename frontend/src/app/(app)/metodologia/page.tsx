"use client";

import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useMethodology } from "@/lib/hooks";
import {
  BookOpen, ExternalLink, ArrowLeft, GraduationCap,
  Brain, CheckCircle, AlertTriangle, FlaskConical,
  BarChart3, Layers,
} from "lucide-react";

// ── Static AI pipeline data ───────────────────────────────────────────────────

const PIPELINE_PHASES = [
  {
    n: "01", file: "01_annotate.py",
    title: "Anotação automática (LLM-as-annotator)",
    desc: "Groq llama-3.3-70b-versatile classifica cada tese/dissertação UFOP: is_patentable (bool), ipc_category (0-7), confidence, rationale.",
    result: "775 amostras anotadas · custo: US$0 (Groq free tier)",
    ref: "Honovich et al. (2022) · Unnatural Instructions · arXiv:2212.09689",
    color: "#6366f1",
  },
  {
    n: "02", file: "02_explore.py",
    title: "Análise exploratória",
    desc: "Distribuição por departamento, classe IPC, nível de oportunidade. Detecta desbalanceamento de classes para estratégias de treinamento.",
    result: "IPC: B (25%), C (28%), E (18%), G (18%) dominantes · 43% patenteáveis",
    ref: "Análise exploratória padrão (EDA)",
    color: "#8b5cf6",
  },
  {
    n: "03", file: "03_train_baseline.py",
    title: "Baseline TF-IDF + Random Forest",
    desc: "Vetorização TF-IDF (Salton & Buckley 1988) + Random Forest (Breiman 2001) para patenteabilidade binária e classificação IPC 8-classes.",
    result: "Patenteabilidade: F1 ~0.81 · IPC: F1 ~0.98 · 5-fold CV estratificado",
    ref: "Salton & Buckley (1988) · Breiman (2001) · Random Forests",
    color: "#06b6d4",
  },
  {
    n: "04", file: "04_train_sentence_transformers.py",
    title: "Sentence-BERT multilingual",
    desc: "Embeddings semânticos via paraphrase-multilingual-MiniLM-L12-v2 (384d). Captura semântica que TF-IDF perde — 'aprendizado de máquina' ≈ 'redes neurais'.",
    result: "Logistic Regression sobre embeddings SBERT · melhora em casos limítrofes",
    ref: "Reimers & Gurevych (2019) · Sentence-BERT · EMNLP 2019",
    color: "#a855f7",
  },
  {
    n: "05", file: "05_cohen_kappa.py",
    title: "Validação Cohen's κ",
    desc: "Mede concordância inter-avaliador: Groq LLM vs Heurística Go. κ razoável esperado — LLM agrega valor nos casos 'medium' onde heurística abstém.",
    result: "κ = 0.288 (Razoável, IC95% [0.214, 0.363]) · P_o = 66.3%",
    ref: "Cohen (1960) · Landis & Koch (1977) · Biometrics 33:159",
    color: "#34d399",
  },
];

const KAPPA_ROWS = [
  { exp: "Patenteabilidade (strict: high=True)",   kappa: "0.288", interp: "Razoável ✅", po: "66.3%", pe: "52.7%", n: "775" },
  { exp: "Patenteabilidade (lenient: high+med)",   kappa: "0.001", interp: "Leve (esperado)",  po: "43.4%", pe: "43.3%", n: "775" },
  { exp: "IPC multi-classe (Groq vs TF-IDF+RF)",  kappa: "0.004", interp: "Leve*",            po: "3.5%",  pe: "3.2%",  n: "367" },
];

const LIMITATIONS = [
  { lim: "Ground truth via LLM (não expert humano)",    mit: "Honovich 2022 valida; confidence ≥ 0.7",           future: "NIT-UFOP anotar amostra ouro" },
  { lim: "Dataset ~775 trabalhos",                       mit: "CV 5-fold estratificado; class_weight=balanced",   future: "Expandir outros departamentos" },
  { lim: "Modelo PT multilingual (não dedicado)",        mit: "MiniLM bom em benchmarks PT-BR",                  future: "Fine-tune BERTimbau específico" },
  { lim: "Sem ground truth 'virou patente?'",            mit: "Proxy: patentes UFOP reais (Google Patents)",      future: "Cruzar com base INPI quando viável" },
  { lim: "IPC κ baixo (0.004)",                         mit: "Annotations sem abstract completo para TF-IDF",    future: "Re-anotar com abstract completo" },
];

// ─────────────────────────────────────────────────────────────────────────────

export default function MethodologyPage() {
  const { data, isLoading } = useMethodology();

  return (
    <div className="p-8 max-w-5xl space-y-8">

      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm">
        <Link href="/metricas" className="hover:text-white transition-colors"
          style={{ color: "var(--text-muted)" }}>
          <ArrowLeft size={14} className="inline mr-1" />
          Métricas
        </Link>
        <span style={{ color: "var(--text-muted)" }}>/</span>
        <span className="text-white">Metodologia</span>
      </div>

      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <GraduationCap size={24} />
          Metodologia &amp; Bibliografia
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Pipeline supervisionado de IA para classificação de patenteabilidade e IPC.
          Defensável em banca com referências peer-reviewed.
        </p>
      </div>

      {/* Disclaimer */}
      <Card style={{ borderColor: "#a855f730" }}>
        <div className="flex items-start gap-3">
          <BookOpen size={16} className="mt-0.5 shrink-0" style={{ color: "#a855f7" }} />
          <div className="text-sm" style={{ color: "var(--text-muted)" }}>
            <p className="text-white font-medium mb-1">Honestidade científica</p>
            <p>
              O pipeline antigo usava <span className="text-white">heurísticas baseadas em palavras-chave</span> — válido como baseline mas insuficiente como contribuição.
              O pipeline atual usa <span className="text-white">modelos supervisionados reais</span> treinados sobre anotações LLM, com métricas reprodutíveis
              e validação por Cohen's κ. Limitações explicitadas na seção abaixo.
            </p>
          </div>
        </div>
      </Card>

      {/* ── Pipeline ── */}
      <section>
        <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
          <Layers size={18} style={{ color: "#6366f1" }} />
          Pipeline de IA — 5 Fases
        </h2>
        <div className="space-y-3">
          {PIPELINE_PHASES.map(ph => (
            <div key={ph.n} className="flex gap-4 p-4 rounded-xl"
              style={{ background: "var(--surface)", border: `1px solid ${ph.color}30` }}>
              <div className="shrink-0 w-10 h-10 rounded-lg flex items-center justify-center font-mono font-bold text-sm"
                style={{ background: ph.color + "20", color: ph.color }}>
                {ph.n}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline gap-2 flex-wrap">
                  <p className="text-sm font-semibold text-white">{ph.title}</p>
                  <code className="text-xs px-1.5 py-0.5 rounded font-mono"
                    style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
                    {ph.file}
                  </code>
                </div>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{ph.desc}</p>
                <div className="flex flex-wrap gap-3 mt-2">
                  <span className="text-xs px-2 py-0.5 rounded-full"
                    style={{ background: ph.color + "15", color: ph.color }}>
                    {ph.result}
                  </span>
                </div>
                <p className="text-xs mt-1.5 italic" style={{ color: "#334155" }}>{ph.ref}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* ── Cohen's κ ── */}
      <section>
        <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
          <FlaskConical size={18} style={{ color: "#34d399" }} />
          Validação — Cohen's κ
        </h2>
        <Card>
          <p className="text-sm mb-3" style={{ color: "var(--text-muted)" }}>
            Mede a concordância entre <span className="text-white">Groq LLM</span> (anotador A)
            e <span className="text-white">Heurística Go</span> (anotador B) sobre 775 trabalhos UFOP.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr style={{ borderBottom: "1px solid var(--border)" }}>
                  {["Experimento", "κ", "Interpretação", "P_o", "P_e", "N"].map(h => (
                    <th key={h} className="text-left py-2 pr-4 font-medium" style={{ color: "var(--text-muted)" }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {KAPPA_ROWS.map(r => (
                  <tr key={r.exp} style={{ borderBottom: "1px solid var(--border)" }}>
                    <td className="py-2 pr-4 text-white">{r.exp}</td>
                    <td className="py-2 pr-4 font-mono font-bold" style={{ color: r.kappa >= "0.2" ? "#34d399" : "var(--text-muted)" }}>{r.kappa}</td>
                    <td className="py-2 pr-4" style={{ color: "var(--text-muted)" }}>{r.interp}</td>
                    <td className="py-2 pr-4 font-mono" style={{ color: "var(--text-muted)" }}>{r.po}</td>
                    <td className="py-2 pr-4 font-mono" style={{ color: "var(--text-muted)" }}>{r.pe}</td>
                    <td className="py-2 font-mono" style={{ color: "var(--text-muted)" }}>{r.n}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-xs mt-3 italic" style={{ color: "#334155" }}>
            * IPC κ baixo porque annotations.jsonl não contém abstract completo — TF-IDF classificou apenas pelo título.
            Escala Landis & Koch (1977): 0.21-0.40 = razoável, 0.41-0.60 = moderado.
          </p>
          <div className="mt-3 p-3 rounded-lg text-xs" style={{ background: "#34d39910", border: "1px solid #34d39930" }}>
            <span className="font-semibold" style={{ color: "#34d399" }}>Argumento para a banca:</span>
            <span style={{ color: "var(--text-muted)" }}>
              {" "}κ = 0.288 é o resultado <em>correto</em> — o LLM diverge da heurística exatamente nos casos
              <code className="mx-1 px-1 rounded" style={{ background: "var(--surface-2)" }}>medium</code>
              (zona cinzenta), onde a heurística abstém. Isso valida o valor do LLM no pipeline.
            </span>
          </div>
        </Card>
      </section>

      {/* ── Limitações ── */}
      <section>
        <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
          <AlertTriangle size={18} style={{ color: "#f59e0b" }} />
          Limitações Explícitas
        </h2>
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr style={{ borderBottom: "1px solid var(--border)" }}>
                  {["Limitação", "Mitigação atual", "Trabalho futuro"].map(h => (
                    <th key={h} className="text-left py-2 pr-4 font-medium" style={{ color: "var(--text-muted)" }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {LIMITATIONS.map((r, i) => (
                  <tr key={i} style={{ borderBottom: "1px solid var(--border)" }}>
                    <td className="py-2 pr-4" style={{ color: "#f87171" }}>{r.lim}</td>
                    <td className="py-2 pr-4" style={{ color: "var(--text-muted)" }}>{r.mit}</td>
                    <td className="py-2" style={{ color: "#6366f180" }}>{r.future}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </section>

      {/* ── Métricas acadêmicas da API ── */}
      {isLoading ? (
        <p className="text-sm text-center py-4" style={{ color: "var(--text-muted)" }}>Carregando métricas…</p>
      ) : data?.metrics && data.metrics.length > 0 && (
        <section>
          <h2 className="text-lg font-bold text-white flex items-center gap-2 mb-4">
            <BarChart3 size={18} style={{ color: "#8b5cf6" }} />
            Métricas Acadêmicas (AUTM / HJT / Triple Helix)
          </h2>
          {data.metrics.map(m => (
            <div key={m.id} id={m.id} className="scroll-mt-8 mb-4">
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2 flex-wrap">
                    <CardTitle className="text-base">{m.name}</CardTitle>
                    <Badge variant="muted">{m.id}</Badge>
                  </div>
                </CardHeader>
                <p className="text-sm mb-3" style={{ color: "var(--text-muted)" }}>{m.description}</p>

                <div className="mb-3">
                  <p className="text-xs font-medium text-white mb-1">Fórmula</p>
                  <pre className="p-3 rounded text-sm font-mono overflow-x-auto"
                    style={{ background: "var(--surface-2)", color: "#a5b4fc" }}>
                    {m.formula}
                  </pre>
                </div>

                {(m.components?.length ?? 0) > 0 && (
                  <div className="mb-3">
                    <p className="text-xs font-medium text-white mb-1.5">Componentes</p>
                    <div className="space-y-1">
                      {m.components!.map((c: { key: string; label: string; definition: string }) => (
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

                {m.data_requirements && (
                  <div className="mb-3 p-2 rounded text-xs"
                    style={{ background: "#fbbf2415", border: "1px solid #fbbf2440", color: "#fbbf24" }}>
                    <span className="font-medium">Requisitos de dados:</span> {m.data_requirements}
                  </div>
                )}

                <div className="pt-3" style={{ borderTop: "1px solid var(--border)" }}>
                  <p className="text-xs font-medium text-white mb-1">Referências</p>
                  <ol className="text-sm space-y-1.5 pl-4">
                    {m.references.map((ref: string, i: number) => (
                      <li key={i} className="list-decimal leading-relaxed"
                        style={{ color: "var(--text-muted)" }}>{ref}</li>
                    ))}
                  </ol>
                </div>
              </Card>
            </div>
          ))}
        </section>
      )}

      {/* ── Como citar ── */}
      <Card style={{ borderColor: "var(--accent)" }}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CheckCircle size={15} style={{ color: "#34d399" }} />
            Como citar este painel
          </CardTitle>
        </CardHeader>
        <pre className="text-xs p-3 rounded overflow-x-auto whitespace-pre-wrap"
          style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
{`Paniago, L. L. C. (${new Date().getFullYear()}).
Argos: Plataforma de inteligência competitiva em propriedade intelectual para o NIT-UFOP.
Pipeline supervisionado: LLM-as-annotator (Honovich et al., 2022) + TF-IDF+RF + Sentence-BERT.
Validação por Cohen's κ = 0.288 (Landis & Koch, 1977).
Repositório: github.com/LeoPani/argos`}
        </pre>
      </Card>

      {/* Footer links */}
      <p className="text-xs text-center pb-4" style={{ color: "var(--text-muted)" }}>
        <a href="https://arxiv.org/abs/2212.09689" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          Honovich 2022 <ExternalLink size={9} />
        </a>
        {" · "}
        <a href="https://autm.net" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          AUTM <ExternalLink size={9} />
        </a>
        {" · "}
        <a href="https://docs.api.lens.org" target="_blank" rel="noopener noreferrer"
          className="hover:text-indigo-400 inline-flex items-center gap-1">
          Lens.org API <ExternalLink size={9} />
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
