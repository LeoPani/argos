"use client";

// /roadmap — visão pública de o que foi feito, o que falta, e por que.
// Espelha ROADMAP.md mas formatado pra exibição. Usado tanto pela banca
// quanto pelo orientador como referência rápida.

import Link from "next/link";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { UnderConstructionBadge } from "@/components/ui/under-construction";
import {
  CheckCircle2, Construction, FileText, Clock,
  AlertCircle, BookOpen, ExternalLink,
} from "lucide-react";

const done = [
  { area: "Núcleo de PI",            count: 8, refs: "Lei 9.279, Lei 10.973" },
  { area: "Indicadores acadêmicos",  count: 10, refs: "AUTM, HJT, Etzkowitz, Pakes, Griliches…" },
  { area: "Frontend",                count: 18, refs: "Next.js App Router + dark theme + ⌘K" },
  { area: "Dados UFOP reais",        count: 60, refs: "OAI-PMH DEDIR + DEMIN testados" },
];

const inProgress = [
  {
    title: "Lens.org integração real",
    why: "Cadastro acadêmico requer email institucional UFOP (1-3 dias aprovação).",
    impact: "Habilita PCI Lanjouw-Schankerman completo + forward citations reais.",
    when: "Quando token chegar — sistema já é dual-mode (mock determinístico funciona).",
  },
  {
    title: "BERT FastAPI em produção",
    why: "Modelo ~440MB não cabe em Vercel free tier. Indicado: Hugging Face Spaces.",
    impact: "Smart Filing terá IPC inference real em vez de fallback heurístico.",
    when: "Após upload do modelo no HF Spaces (~30 min).",
  },
  {
    title: "INPI bulk ingestion",
    why: "Worker já existe; depende de BERT em produção + 30min processamento por RPI.",
    impact: "+1000-3000 patentes BR reais por RPI semanal.",
    when: "Imediato após BERT estar online.",
  },
];

const futureFases = [
  {
    title: "Blockchain timestamping (Phase 4)",
    why: "Decisão explícita: adiar para após validação acadêmica.",
    technical: "Hash on-chain Polygon (~$0.01/tx) de disputas + provas.",
    status: "UI tem placeholders, nada conectado.",
  },
  {
    title: "Integração Lattes",
    why: "LGPD: dados pessoais. Plataforma sem API pública, scraping violaria ToS.",
    technical: "Mitigação: /inventors/[name] usa metadata do repositório OAI público.",
    status: "Substituído por agregação OAI.",
  },
  {
    title: "Web of Science / Scopus",
    why: "Assinaturas pagas, acessíveis apenas via VPN institucional UFOP.",
    technical: "Possível via CAPES/Periódicos com autenticação SAML.",
    status: "Documentado para Phase 6.",
  },
  {
    title: "Worker INPI proativo (cron)",
    why: "Operacional (infra de cron + retry), não acadêmico.",
    technical: "Agendamento semanal coincidindo com publicação RPI.",
    status: "Phase 7 — após deploy em produção.",
  },
];

export default function RoadmapPage() {
  return (
    <div className="p-8 max-w-5xl space-y-6 fade-in">
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <BookOpen size={22} />
          Roadmap &amp; Justificativas
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Mapa do que foi implementado, do que está em construção, e do que ficou
          documentado pra fases futuras — com justificativa para defesa acadêmica.
        </p>
      </div>

      {/* Done */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CheckCircle2 size={16} className="text-emerald-400" />
            Implementado e em produção
          </CardTitle>
        </CardHeader>
        <div className="grid grid-cols-2 gap-3">
          {done.map(d => (
            <div key={d.area} className="p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid #34d39940" }}>
              <div className="flex items-center justify-between mb-1">
                <p className="text-sm font-semibold text-white">{d.area}</p>
                <Badge variant="success">{d.count}</Badge>
              </div>
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>{d.refs}</p>
            </div>
          ))}
        </div>
        <div className="mt-4 pt-3" style={{ borderTop: "1px solid var(--border)" }}>
          <Link href="/metodologia" className="text-xs text-indigo-400 hover:text-indigo-300">
            Ver bibliografia completa de indicadores <ExternalLink size={9} className="inline" />
          </Link>
        </div>
      </Card>

      {/* In progress */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Construction size={16} className="text-amber-400" />
            Em construção <UnderConstructionBadge />
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          {inProgress.map(ip => (
            <div key={ip.title} className="p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid #fbbf2440" }}>
              <p className="text-sm font-semibold text-white">{ip.title}</p>
              <div className="grid grid-cols-3 gap-3 mt-2 text-xs">
                <Field label="Por que falta"  value={ip.why} />
                <Field label="Impacto"        value={ip.impact} />
                <Field label="Quando ativar"  value={ip.when} />
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Future phases */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock size={16} className="text-indigo-400" />
            Documentado para fases futuras
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          {futureFases.map(f => (
            <div key={f.title} className="p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
              <p className="text-sm font-semibold text-white mb-1">{f.title}</p>
              <div className="space-y-1 text-xs" style={{ color: "var(--text-muted)" }}>
                <p><span className="text-white">Por que adiado:</span> {f.why}</p>
                <p><span className="text-white">Solução técnica:</span> {f.technical}</p>
                <p><span className="text-white">Status atual:</span> {f.status}</p>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* Defesa */}
      <Card style={{ borderColor: "#a855f730" }}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertCircle size={14} className="text-purple-400" />
            Para o slideshow do orientador
          </CardTitle>
        </CardHeader>
        <ol className="space-y-1.5 text-sm pl-5 list-decimal" style={{ color: "var(--text-muted)" }}>
          <li><span className="text-white">Problema:</span> NIT-UFOP sem ferramenta de inteligência de PI</li>
          <li><span className="text-white">Estado da arte:</span> AUTM, Etzkowitz, HJT, Lens.org</li>
          <li><span className="text-white">Solução proposta:</span> Argos — 10 indicadores peer-reviewed + automação</li>
          <li><span className="text-white">Implementação:</span> Go + Postgres + Next.js + BERT</li>
          <li><span className="text-white">Resultados:</span> Métricas computadas com dados UFOP reais</li>
          <li><span className="text-white">Limitações:</span> ver seção &quot;em construção&quot; acima</li>
          <li><span className="text-white">Trabalhos futuros:</span> Phases 4-7 documentadas</li>
        </ol>
      </Card>

      <div className="text-xs text-center p-3 rounded"
        style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
        <FileText size={11} className="inline mr-1" />
        Documento técnico completo: <code className="text-indigo-400">ROADMAP.md</code> na raiz do repo.
      </div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium text-white mb-0.5">{label}</p>
      <p style={{ color: "var(--text-muted)" }}>{value}</p>
    </div>
  );
}
