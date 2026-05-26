"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import {
  ShieldCheck, Hash, Link2, Clock, Plus, ChevronDown,
  ChevronUp, Copy, CheckCheck, AlertTriangle, Loader2,
  FileText,
} from "lucide-react";
import { useTimestamps } from "@/lib/hooks";
import { formatDate } from "@/lib/utils";
import type { IPTimestamp } from "@/lib/types";

// ─── helpers ──────────────────────────────────────────────────────────────────

function shortHash(h: string) {
  return h ? `${h.slice(0, 8)}…${h.slice(-8)}` : "—";
}

const CATEGORY_OPTIONS = [
  { value: "invenção",              label: "Patente de Invenção" },
  { value: "modelo de utilidade",   label: "Modelo de Utilidade" },
  { value: "software",              label: "Software / Algoritmo" },
  { value: "design",                label: "Desenho Industrial" },
  { value: "segredo industrial",    label: "Segredo Industrial" },
];

// ─── Certificado ─────────────────────────────────────────────────────────────

function Certificate({ record, canonical }: { record: IPTimestamp; canonical?: string }) {
  const [copied, setCopied] = useState(false);

  function copyHash() {
    navigator.clipboard.writeText(record.content_hash);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="rounded-xl border-2 p-6 space-y-4"
      style={{ borderColor: "#34d399", background: "#0d1f18" }}>
      {/* Header */}
      <div className="flex items-center gap-3">
        <ShieldCheck size={28} className="text-emerald-400 shrink-0" />
        <div>
          <p className="text-xs text-emerald-400 font-mono uppercase tracking-widest">
            Certificado de Anterioridade — Argos/UFOP
          </p>
          <p className="text-lg font-bold text-white mt-0.5">{record.title}</p>
        </div>
      </div>

      <hr style={{ borderColor: "#1e3a2e" }} />

      {/* Grid de dados */}
      <div className="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>Categoria</p>
          <p className="text-white capitalize">{record.category}</p>
        </div>
        <div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>Inventores</p>
          <p className="text-white">{record.authors?.join(", ") || "—"}</p>
        </div>
        <div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>Timestamp UTC</p>
          <p className="text-white font-mono text-xs">
            {new Date(record.created_at).toISOString()}
          </p>
        </div>
        <div>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>Posição na cadeia</p>
          <p className="text-white font-mono">#{record.chain_index}</p>
        </div>
      </div>

      {/* Hash principal */}
      <div className="rounded-lg p-3 space-y-1" style={{ background: "#0a1a10" }}>
        <div className="flex items-center justify-between">
          <span className="text-xs text-emerald-400 font-mono flex items-center gap-1">
            <Hash size={11} /> SHA-256 · Conteúdo
          </span>
          <button onClick={copyHash}
            className="text-xs flex items-center gap-1 text-slate-400 hover:text-emerald-400 transition-colors">
            {copied ? <><CheckCheck size={11} /> copiado</> : <><Copy size={11} /> copiar</>}
          </button>
        </div>
        <p className="font-mono text-xs text-emerald-300 break-all">{record.content_hash}</p>
      </div>

      {/* Hash anterior (cadeia) */}
      {record.prev_hash && (
        <div className="rounded-lg p-3 space-y-1" style={{ background: "#0a1a10" }}>
          <span className="text-xs text-slate-400 font-mono flex items-center gap-1">
            <Link2 size={11} /> Hash anterior (cadeia)
          </span>
          <p className="font-mono text-xs text-slate-400 break-all">{record.prev_hash}</p>
        </div>
      )}

      {/* Conteúdo canônico */}
      {canonical && (
        <div className="rounded-lg p-3 space-y-1" style={{ background: "#0a1a10" }}>
          <span className="text-xs text-slate-400 font-mono">Conteúdo hasheado (canônico)</span>
          <p className="font-mono text-xs text-slate-500 break-all">{canonical}</p>
        </div>
      )}

      <p className="text-[10px] text-slate-600 leading-relaxed">
        Este certificado é gerado pelo sistema Argos (UFOP) e usa SHA-256 para criar
        prova criptográfica de existência do conteúdo na data indicada.
        Não substitui depósito de patente no INPI, mas estabelece anterioridade documental
        auditável para fins de defesa de prior art.
      </p>
    </div>
  );
}

// ─── Item da lista ────────────────────────────────────────────────────────────

function TimestampRow({ record }: { record: IPTimestamp }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="rounded-lg border" style={{ borderColor: "var(--border)", background: "var(--surface-1)" }}>
      <button
        className="w-full flex items-center justify-between p-4 text-left"
        onClick={() => setOpen(o => !o)}>
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
            style={{ background: "#1a2a20" }}>
            <span className="text-xs text-emerald-400 font-mono">#{record.chain_index}</span>
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium text-white truncate">{record.title}</p>
            <p className="text-xs mt-0.5 flex items-center gap-2" style={{ color: "var(--text-muted)" }}>
              <Clock size={10} />
              {formatDate(record.created_at)}
              <span>·</span>
              <span className="font-mono">{shortHash(record.content_hash)}</span>
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-xs px-2 py-0.5 rounded-full capitalize"
            style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
            {record.category}
          </span>
          {open ? <ChevronUp size={14} className="text-slate-500" /> : <ChevronDown size={14} className="text-slate-500" />}
        </div>
      </button>
      {open && (
        <div className="px-4 pb-4">
          <Certificate record={record} />
        </div>
      )}
    </div>
  );
}

// ─── Formulário ───────────────────────────────────────────────────────────────

function CreateForm({ onCreated }: { onCreated: (r: IPTimestamp, canonical: string) => void }) {
  const [title, setTitle]       = useState("");
  const [desc, setDesc]         = useState("");
  const [authors, setAuthors]   = useState("");
  const [category, setCategory] = useState("invenção");
  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.timestamps.create({
        title: title.trim(),
        description: desc.trim(),
        authors: authors.split(",").map(a => a.trim()).filter(Boolean),
        category,
      });
      onCreated(res, res.canonical_content ?? "");
      setTitle(""); setDesc(""); setAuthors(""); setCategory("invenção");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao registrar");
    } finally {
      setLoading(false);
    }
  }

  const inputCls = "w-full rounded-lg px-3 py-2 text-sm text-white outline-none focus:ring-1 focus:ring-emerald-500";
  const inputStyle = { background: "var(--surface-2)", border: "1px solid var(--border)" };

  return (
    <form onSubmit={submit} className="space-y-4">
      <div>
        <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
          Título da invenção *
        </label>
        <input
          className={inputCls} style={inputStyle}
          placeholder="Ex: Processo de flotação reversa com reagentes biodegradáveis"
          value={title} onChange={e => setTitle(e.target.value)}
          required
        />
      </div>

      <div>
        <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
          Descrição / Resumo técnico
        </label>
        <textarea
          className={inputCls} style={{ ...inputStyle, resize: "vertical" }}
          rows={4}
          placeholder="Descreva o conceito técnico, o problema que resolve e a solução proposta..."
          value={desc} onChange={e => setDesc(e.target.value)}
        />
        <p className="text-[10px] mt-1" style={{ color: "var(--text-muted)" }}>
          Quanto mais detalhada, mais forte a prova de anterioridade.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
            Inventores (vírgula para múltiplos)
          </label>
          <input
            className={inputCls} style={inputStyle}
            placeholder="Nome Sobrenome, Outro Inventor"
            value={authors} onChange={e => setAuthors(e.target.value)}
          />
        </div>
        <div>
          <label className="text-xs mb-1 block" style={{ color: "var(--text-muted)" }}>
            Categoria
          </label>
          <select
            className={inputCls} style={inputStyle}
            value={category} onChange={e => setCategory(e.target.value)}>
            {CATEGORY_OPTIONS.map(o => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 rounded-lg text-sm text-red-400"
          style={{ background: "#2a1010", border: "1px solid #ef444440" }}>
          <AlertTriangle size={14} /> {error}
        </div>
      )}

      <Button type="submit" disabled={loading || !title.trim()} className="w-full">
        {loading
          ? <><Loader2 size={14} className="animate-spin" /> Gerando hash…</>
          : <><ShieldCheck size={14} /> Registrar anterioridade</>
        }
      </Button>
    </form>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function AnterioridadePage() {
  const { data, mutate } = useTimestamps();
  const [newRecord, setNewRecord] = useState<IPTimestamp | null>(null);
  const [newCanonical, setNewCanonical] = useState("");

  const records: IPTimestamp[] = data?.items ?? [];
  const total: number = data?.pagination?.total ?? 0;

  function handleCreated(r: IPTimestamp, canonical: string) {
    setNewRecord(r);
    setNewCanonical(canonical);
    mutate();
  }

  return (
    <div className="p-8 space-y-6 max-w-4xl mx-auto">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <ShieldCheck size={22} className="text-emerald-400" />
          Registro de Anterioridade
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Gera prova criptográfica de existência (SHA-256) com cadeia de hashes auditável.
          Estabelece anterioridade documental para prior art sem depósito no INPI.
        </p>
      </div>

      {/* Como funciona */}
      <Card>
        <div className="grid grid-cols-3 gap-4 text-center">
          {[
            { icon: <FileText size={18} className="text-blue-400" />, title: "1. Descreva", desc: "Informe título, descrição técnica e inventores" },
            { icon: <Hash size={18} className="text-purple-400" />, title: "2. Hash SHA-256", desc: "Sistema gera hash criptográfico do conteúdo + timestamp UTC" },
            { icon: <Link2 size={18} className="text-emerald-400" />, title: "3. Cadeia", desc: "Cada registro encadeia o anterior — adulteração retroativa é detectável" },
          ].map(s => (
            <div key={s.title} className="space-y-2">
              <div className="flex justify-center">{s.icon}</div>
              <p className="text-sm font-semibold text-white">{s.title}</p>
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>{s.desc}</p>
            </div>
          ))}
        </div>
      </Card>

      <div className="grid grid-cols-5 gap-6">
        {/* Formulário */}
        <div className="col-span-3 space-y-4">
          <Card>
            <h2 className="text-sm font-semibold text-white mb-4 flex items-center gap-2">
              <Plus size={14} /> Novo registro
            </h2>
            <CreateForm onCreated={handleCreated} />
          </Card>
        </div>

        {/* Certificado recém gerado */}
        <div className="col-span-2">
          {newRecord ? (
            <div className="space-y-3">
              <p className="text-xs font-semibold text-emerald-400 flex items-center gap-1">
                <CheckCheck size={12} /> Certificado gerado
              </p>
              <Certificate record={newRecord} canonical={newCanonical} />
            </div>
          ) : (
            <div className="h-full flex items-center justify-center rounded-xl border-2 border-dashed"
              style={{ borderColor: "var(--border)", minHeight: 200 }}>
              <div className="text-center space-y-2 p-6">
                <ShieldCheck size={32} className="mx-auto text-slate-700" />
                <p className="text-sm" style={{ color: "var(--text-muted)" }}>
                  O certificado aparece aqui após o registro
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Histórico */}
      <div>
        <h2 className="text-sm font-semibold text-white mb-3">
          Cadeia de registros
          <span className="ml-2 font-normal text-xs" style={{ color: "var(--text-muted)" }}>
            · {total} total
          </span>
        </h2>
        {records.length === 0 ? (
          <div className="text-center py-12" style={{ color: "var(--text-muted)" }}>
            <Hash size={32} className="mx-auto mb-3 text-slate-700" />
            <p className="text-sm">Nenhum registro ainda. Crie o primeiro acima.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {records.map(r => <TimestampRow key={r.id} record={r} />)}
          </div>
        )}
      </div>
    </div>
  );
}
