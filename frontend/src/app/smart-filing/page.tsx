"use client";

// /smart-filing — assistente acadêmico que ajuda o NIT-UFOP a avaliar
// uma invenção antes do depósito INPI.
//
// Fluxo do wizard:
//   1. Inventor preenche título + abstract + área
//   2. Backend roda BERT (IPC) + busca prior art interno (ILIKE)
//   3. UI mostra: score composto, IPC sugerido, top 5 hits, claim template

import { useState } from "react";
import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { MetricTooltip } from "@/components/ui/metric-tooltip";
import { useToast } from "@/components/ui/toast";
import { api } from "@/lib/api";
import { ipcLabel } from "@/lib/utils";
import type { FilingSuggestion } from "@/lib/types";
import {
  Sparkles, ArrowLeft, BookOpen, Lightbulb, AlertCircle,
  CheckCircle2, FileText, RefreshCw, X, ExternalLink, Zap, Copy,
} from "lucide-react";

const SAMPLE_FILINGS = [
  {
    label: "Lítio (química)",
    title: "Processo eletroquímico de extração seletiva de lítio em pegmatitos da região de Araçuaí",
    abstract: "Método hidrometalúrgico que combina lixiviação ácida com eletrodeposição assistida por membrana de troca iônica para obtenção de carbonato de lítio com pureza superior a 99,5% e consumo energético 30% menor que processos convencionais. O processo elimina o uso de reagentes orgânicos e gera efluentes recicláveis.",
  },
  {
    label: "Biossensor",
    title: "Biossensor eletroquímico de baixo custo para detecção rápida de E. coli em água potável",
    abstract: "Dispositivo portátil baseado em eletrodos de grafeno funcionalizados com aptâmeros específicos para detecção de Escherichia coli em menos de 5 minutos, com limite de detecção em ppb e sensibilidade clinicamente relevante.",
  },
  {
    label: "Microrredes",
    title: "Sistema de controle adaptativo para otimização de energia em microrredes rurais isoladas",
    abstract: "Algoritmo embarcado em microcontrolador de baixo consumo que aprende padrões de geração solar e consumo doméstico para minimizar perdas, reduzir uso de bateria e aumentar disponibilidade energética em comunidades sem acesso à rede principal.",
  },
];

export default function SmartFilingPage() {
  const toast = useToast();
  const [title, setTitle]       = useState("");
  const [abstract, setAbstract] = useState("");
  const [field, setField]       = useState("");
  const [busy, setBusy]         = useState(false);
  const [result, setResult]     = useState<FilingSuggestion | null>(null);
  const [error, setError]       = useState<string | null>(null);

  function loadSample(s: typeof SAMPLE_FILINGS[0]) {
    setTitle(s.title);
    setAbstract(s.abstract);
    setField("Pesquisa UFOP");
    setResult(null);
    setError(null);
  }

  async function analyze(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    setBusy(true); setError(null); setResult(null);
    try {
      const r = await api.smartFiling.analyze({ title, abstract, field });
      setResult(r);
      toast.success("Análise concluída", `Recomendação: ${r.recommendation.toUpperCase()}`);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Erro desconhecido";
      setError(msg);
      toast.error("Falha na análise", msg);
    } finally { setBusy(false); }
  }

  function reset() {
    setTitle(""); setAbstract(""); setField(""); setResult(null); setError(null);
  }

  return (
    <div className="p-8 max-w-5xl space-y-6 fade-in">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm">
        <Link href="/metricas" className="hover:text-white transition-colors"
          style={{ color: "var(--text-muted)" }}>
          <ArrowLeft size={14} className="inline mr-1" />
          Métricas
        </Link>
        <span style={{ color: "var(--text-muted)" }}>/</span>
        <span className="text-white">Smart Filing Assistant</span>
      </div>

      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Sparkles size={22} className="text-purple-400" />
          Smart Filing Assistant
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Avalie a patenteabilidade de um draft de invenção <span className="text-white">antes</span> do depósito no INPI.
          IPC via BERT (quando online); senão heurística por keywords. Anterioridade via Jaccard sobre portfolio UFOP local.
        </p>
      </div>

      {/* Disclaimer */}
      <Card style={{ borderColor: "#fbbf2430" }}>
        <div className="flex items-start gap-3">
          <BookOpen size={15} className="text-amber-400 mt-0.5 shrink-0" />
          <div className="text-xs" style={{ color: "var(--text-muted)" }}>
            <p className="text-white font-medium mb-1">Honestidade metodológica</p>
            Score composto baseado em Bessen (2008) <em>Research Policy</em> + Lerner-Seru (2017) <em>RFS</em>.
            <strong className="text-white"> O score atual é heurístico</strong> — quando o classificador
            treinado entrar no ar (ver banner no dashboard), este card vira ML supervisionado.
            Anterioridade é <em>local</em> (só portfolio UFOP), não substitui busca no INPI/Espacenet.{" "}
            <Link href="/metodologia" className="text-indigo-400 hover:text-indigo-300">
              Ver metodologia completa <ExternalLink size={9} className="inline" />
            </Link>
          </div>
        </div>
      </Card>

      {/* Form */}
      <Card>
        <CardHeader>
          <CardTitle>Draft da invenção</CardTitle>
          <div className="flex gap-2">
            {SAMPLE_FILINGS.map(s => (
              <Button key={s.label} variant="ghost" size="sm" onClick={() => loadSample(s)}>
                {s.label}
              </Button>
            ))}
          </div>
        </CardHeader>

        <form onSubmit={analyze} className="space-y-3">
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
              Título da invenção *
            </label>
            <input
              value={title}
              onChange={e => setTitle(e.target.value)}
              required
              placeholder="ex: Processo eletroquímico de extração de lítio…"
              className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
            />
          </div>

          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
              Abstract / descrição técnica
              <span className="ml-2 text-xs" style={{ color: "var(--text-muted)" }}>
                (mín. 200 chars recomendado · atual: {abstract.length})
              </span>
            </label>
            <textarea
              value={abstract}
              onChange={e => setAbstract(e.target.value)}
              rows={6}
              placeholder="Descreva o método, os efeitos técnicos, as vantagens sobre o estado da arte…"
              className="w-full px-4 py-3 rounded-lg text-sm outline-none resize-y"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white", fontFamily: "inherit" }}
            />
          </div>

          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
              Área de pesquisa (opcional)
            </label>
            <input
              value={field}
              onChange={e => setField(e.target.value)}
              placeholder="ex: Hidrometalurgia, Biossensores, Microeletrônica…"
              className="w-full px-4 py-2.5 rounded-lg text-sm outline-none"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "white" }}
            />
          </div>

          {error && (
            <div className="p-2 rounded text-xs"
              style={{ background: "#7f1d1d20", border: "1px solid #ef444460", color: "#fca5a5" }}>
              {error}
            </div>
          )}

          <div className="flex gap-2">
            <Button type="submit" size="sm" disabled={busy || !title.trim()}>
              {busy
                ? <><div className="w-3 h-3 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Analisando…</>
                : <><Sparkles size={13} /> Analisar com IA</>}
            </Button>
            {(result || title) && (
              <Button type="button" variant="ghost" size="sm" onClick={reset}>
                <X size={13} /> Limpar
              </Button>
            )}
          </div>
        </form>
      </Card>

      {/* Result */}
      {result && <SuggestionResult sug={result} />}
    </div>
  );
}

// ─── Result component ───────────────────────────────────────────────────────

function SuggestionResult({ sug }: { sug: FilingSuggestion }) {
  const recMeta = {
    proceed:          { icon: CheckCircle2, label: "Prosseguir com depósito", color: "#34d399" },
    refine:           { icon: RefreshCw,    label: "Refinar antes",            color: "#fbbf24" },
    not_recommended:  { icon: AlertCircle,  label: "Não recomendado",          color: "#ef4444" },
  }[sug.recommendation];
  const Icon = recMeta.icon;

  return (
    <div className="space-y-4 fade-in">
      {/* Composite score banner */}
      <Card style={{ borderColor: recMeta.color + "40" }}>
        <div className="flex items-start gap-4">
          <div className="p-3 rounded-xl shrink-0"
            style={{ background: recMeta.color + "20", border: `1px solid ${recMeta.color}40` }}>
            <Icon size={24} style={{ color: recMeta.color }} />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <Badge variant="muted">{sug.methodology}</Badge>
            </div>
            <p className="text-xl font-bold" style={{ color: recMeta.color }}>
              {recMeta.label}
            </p>
            <p className="text-sm mt-0.5" style={{ color: "var(--text-muted)" }}>
              Score composto: <span className="text-white font-semibold">{sug.overall_score}/100</span>
            </p>
          </div>

          {/* IPC */}
          {sug.ipc_letter && (
            <div className="text-right shrink-0">
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>IPC sugerido (BERT)</p>
              <p className="text-2xl font-bold text-white">{sug.ipc_letter}</p>
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>{sug.ipc_name}</p>
              <Badge variant={sug.ipc_confidence === "high" ? "success" : "warning"}>
                {sug.ipc_confidence}
              </Badge>
            </div>
          )}
        </div>

        {/* Component scores */}
        <div className="grid grid-cols-3 gap-3 mt-4">
          <ScoreBox label="Distintividade"  value={sug.distinctiveness} tip="Variedade lexical do título (HJT-light)" />
          <ScoreBox label="Especificidade"  value={sug.specificity}     tip="Riqueza do abstract (Bessen 2008)" />
          <ScoreBox label="Novidade"        value={sug.novelty_score}   tip="1 − max(similaridade com prior art interno)" />
        </div>
      </Card>

      {/* Prior art hits */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText size={14} className="text-indigo-400" />
            Prior art interno ({sug.prior_art_hits.length} hits)
          </CardTitle>
        </CardHeader>

        {sug.prior_art_hits.length === 0 ? (
          <p className="text-sm py-2" style={{ color: "var(--text-muted)" }}>
            ✓ Nenhuma patente similar no portfolio UFOP.
          </p>
        ) : (
          <div className="space-y-2">
            {sug.prior_art_hits.map(h => (
              <Link key={h.patent_id} href={`/patents/${h.patent_id}`}>
                <div className="flex items-center gap-3 p-2.5 rounded transition-colors cursor-pointer hover:bg-white/5"
                  style={{ background: "var(--surface-2)" }}>
                  <SimilarityBar pct={h.similarity_pct} />
                  <span className="font-mono text-xs text-indigo-400 shrink-0">{h.application_number}</span>
                  <span className="text-sm text-white flex-1 truncate">{h.title}</span>
                  {h.ipc_category >= 0 && (
                    <Badge variant="muted">{ipcLabel(h.ipc_category)}</Badge>
                  )}
                  <Badge variant={h.status === "classified" ? "success" : "warning"}>{h.status}</Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </Card>

      {/* Suggested claim */}
      <ClaimCard claim={sug.suggested_claim} />

      {/* Next steps */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CheckCircle2 size={14} className="text-emerald-400" />
            Próximos passos
          </CardTitle>
        </CardHeader>
        <ol className="space-y-1.5 pl-4">
          {sug.next_steps.map((s, i) => (
            <li key={i} className="list-decimal text-sm leading-relaxed text-white">
              {s}
            </li>
          ))}
        </ol>
      </Card>
    </div>
  );
}

function ScoreBox({ label, value, tip }: { label: string; value: number; tip: string }) {
  const color = value >= 70 ? "#34d399" : value >= 45 ? "#fbbf24" : "#f87171";
  return (
    <div className="p-3 rounded-lg" style={{ background: "var(--surface-2)" }}>
      <div className="flex items-center gap-1 text-xs mb-1" style={{ color: "var(--text-muted)" }}>
        {label}
        <span title={tip}>
          <MetricTooltip metricID="autm_health_score" />
        </span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <p className="text-2xl font-bold" style={{ color }}>{value.toFixed(1)}</p>
        <span className="text-xs" style={{ color: "var(--text-muted)" }}>/100</span>
      </div>
      <div className="h-1 rounded-full mt-2" style={{ background: "var(--border)" }}>
        <div className="h-full rounded-full" style={{ width: `${value}%`, background: color }} />
      </div>
    </div>
  );
}

function SimilarityBar({ pct }: { pct: number }) {
  const color = pct >= 70 ? "#ef4444" : pct >= 40 ? "#fbbf24" : "#34d399";
  return (
    <div className="shrink-0 w-12 text-right">
      <p className="text-sm font-mono font-semibold" style={{ color }}>{pct}%</p>
    </div>
  );
}

// ─── ClaimCard — renders Groq claim with visual hierarchy ─────────────────────

function ClaimCard({ claim }: { claim: string }) {
  const isGroq = claim.includes("REIVINDICAÇÃO 1 (independente)");
  const hasArt10 = claim.includes("ART. 10 LPI");

  // Split into sections on lines starting with REIVINDICAÇÃO or ⚠️
  const sections = parseClaimSections(claim);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Lightbulb size={14} className="text-amber-400" />
          {isGroq ? "Reivindicação gerada por IA" : "Template de reivindicação (rascunho)"}
        </CardTitle>
        <div className="flex items-center gap-2">
          {isGroq && (
            <span className="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
              style={{ background: "#7c3aed20", border: "1px solid #7c3aed60", color: "#a78bfa" }}>
              <Zap size={10} />
              Groq llama-3.3-70b
            </span>
          )}
          <Button variant="ghost" size="sm"
            onClick={() => navigator.clipboard.writeText(claim)}>
            <Copy size={11} /> Copiar
          </Button>
        </div>
      </CardHeader>

      {/* Structured rendering */}
      <div className="space-y-3">
        {sections.map((sec, i) => (
          <ClaimSection key={i} section={sec} />
        ))}
      </div>

      {/* Methodological note */}
      <p className="text-xs mt-4 pt-3" style={{ borderTop: "1px solid var(--border)", color: "var(--text-muted)" }}>
        {isGroq
          ? "Gerado por LLM — revise com um agente de PI antes do depósito. Não constitui aconselhamento jurídico."
          : "Template estrutural — baseado em Diretrizes INPI 2023. Revisar com agente antes do depósito."}
        {hasArt10 && (
          <span className="ml-2 text-amber-400 font-medium">
            ⚠ Contém alertas Art. 10 LPI — ver abaixo.
          </span>
        )}
      </p>
    </Card>
  );
}

type ClaimSection = { kind: "claim" | "alert" | "other"; title: string; body: string };

function parseClaimSections(raw: string): ClaimSection[] {
  const lines = raw.split("\n");
  const sections: ClaimSection[] = [];
  let cur: ClaimSection | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      if (cur) cur.body += "\n";
      continue;
    }

    // Section header patterns
    if (/^REIVINDICAÇÃO\s+\d+/i.test(trimmed)) {
      if (cur) sections.push(cur);
      cur = { kind: "claim", title: trimmed, body: "" };
      continue;
    }
    if (/^⚠️?\s*ALERTA/i.test(trimmed) || /^ALERTA\s+ART\./i.test(trimmed)) {
      if (cur) sections.push(cur);
      cur = { kind: "alert", title: trimmed, body: "" };
      continue;
    }
    // Legacy template header
    if (/^REIVINDICAÇÕES\s+(DEPENDENTES|INDEPENDENTES)/i.test(trimmed)) {
      if (cur) sections.push(cur);
      cur = { kind: "claim", title: trimmed, body: "" };
      continue;
    }
    // Notes / NOTA lines
    if (/^NOTA:/i.test(trimmed)) {
      if (cur) sections.push(cur);
      cur = { kind: "other", title: "Nota", body: trimmed };
      continue;
    }

    if (cur) {
      cur.body += (cur.body ? "\n" : "") + trimmed;
    } else {
      cur = { kind: "other", title: "", body: trimmed };
    }
  }
  if (cur && (cur.title || cur.body.trim())) sections.push(cur);
  return sections.filter(s => s.body.trim() || s.title.trim());
}

function ClaimSection({ section }: { section: ClaimSection }) {
  if (section.kind === "alert") {
    return (
      <div className="p-3 rounded-lg" style={{ background: "#78350f20", border: "1px solid #f59e0b40" }}>
        <p className="text-xs font-semibold text-amber-400 mb-1.5">{section.title}</p>
        <div className="space-y-1">
          {section.body.split("\n").filter(Boolean).map((l, i) => (
            <p key={i} className="text-xs" style={{ color: "#fde68a" }}>{l}</p>
          ))}
        </div>
      </div>
    );
  }

  if (section.kind === "claim") {
    const isIndependent = /independente/i.test(section.title);
    const titleColor = isIndependent ? "#a78bfa" : "#94a3b8";
    const borderColor = isIndependent ? "#7c3aed40" : "var(--border)";
    return (
      <div className="p-3 rounded-lg"
        style={{ background: "var(--surface-2)", border: `1px solid ${borderColor}` }}>
        <p className="text-xs font-semibold mb-2" style={{ color: titleColor }}>
          {section.title}
        </p>
        <div className="space-y-1">
          {section.body.split("\n").filter(Boolean).map((l, i) => {
            const isLettered = /^[a-z]\)/.test(l);
            return (
              <p key={i} className="text-sm leading-relaxed"
                style={{ color: isLettered ? "var(--text)" : "var(--text-muted)", paddingLeft: isLettered ? "0.75rem" : 0 }}>
                {l}
              </p>
            );
          })}
        </div>
      </div>
    );
  }

  // "other" / notes
  return (
    <p className="text-xs italic" style={{ color: "var(--text-muted)" }}>
      {section.body}
    </p>
  );
}
