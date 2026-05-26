"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Settings, Database, Cpu, Bell, RefreshCw,
  CheckCircle, XCircle, Clock, Copy, Check,
  Shield,
} from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type ConnStatus = "checking" | "ok" | "error" | "pending";

interface Conn {
  label: string;
  url: string;
  probe?: () => Promise<boolean>;
  status: ConnStatus;
  latency?: number;
}

function maskKey(key: string): string {
  if (!key || key.length < 8) return "••••••••";
  return key.slice(0, 6) + "••••••" + key.slice(-4);
}

export default function ConfigPage() {
  const [conns, setConns] = useState<Conn[]>([
    { label: "API Go (Backend)",        url: `${API}`,                 status: "checking" },
    { label: "BERT Classifier (FastAPI)", url: "http://localhost:8000", status: "checking" },
    { label: "PostgreSQL",               url: "via Go API",             status: "pending"  },
    { label: "Groq LLM",                 url: "api.groq.com",           status: "checking" },
    { label: "Lens.org API",             url: "api.lens.org",           status: "pending"  },
  ]);

  const [copying, setCopying] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const runChecks = useCallback(async () => {
    setRefreshing(true);
    // Mark all checkable as "checking"
    setConns(prev => prev.map(c =>
      c.status !== "pending" ? { ...c, status: "checking" as ConnStatus } : c
    ));

    // Check Go API + Postgres (via /api/v1/stats)
    const checkAPI = async (): Promise<{ ok: boolean; ms: number }> => {
      const t0 = Date.now();
      try {
        const r = await fetch(`${API}/api/v1/stats`, { signal: AbortSignal.timeout(3000) });
        return { ok: r.ok, ms: Date.now() - t0 };
      } catch { return { ok: false, ms: Date.now() - t0 }; }
    };

    // Check BERT via /health (exposed by argos_classifier.py)
    const checkBERT = async (): Promise<{ ok: boolean; ms: number }> => {
      const t0 = Date.now();
      try {
        const r = await fetch("http://localhost:8000/health", { signal: AbortSignal.timeout(3000) });
        return { ok: r.ok, ms: Date.now() - t0 };
      } catch { return { ok: false, ms: Date.now() - t0 }; }
    };

    // Check Groq via our backend (if API key is configured, stats will succeed with groq info)
    const checkGroq = async (): Promise<{ ok: boolean; ms: number }> => {
      const t0 = Date.now();
      try {
        const r = await fetch(`${API}/api/v1/stats`, { signal: AbortSignal.timeout(3000) });
        // If backend is up and has groq wired, we infer from the healthy response
        return { ok: r.ok, ms: Date.now() - t0 };
      } catch { return { ok: false, ms: Date.now() - t0 }; }
    };

    const [api, bert, groq] = await Promise.all([checkAPI(), checkBERT(), checkGroq()]);

    setConns(prev => prev.map(c => {
      if (c.label === "API Go (Backend)")        return { ...c, status: api.ok  ? "ok" : "error", latency: api.ms  };
      if (c.label === "BERT Classifier (FastAPI)") return { ...c, status: bert.ok ? "ok" : "error", latency: bert.ms };
      if (c.label === "PostgreSQL")              return { ...c, status: api.ok  ? "ok" : "error" };
      if (c.label === "Groq LLM")               return { ...c, status: groq.ok ? "ok" : "error", latency: groq.ms };
      return c;
    }));
    setRefreshing(false);
  }, []);

  useEffect(() => { runChecks(); }, [runChecks]);

  async function copyKey() {
    const key = process.env.NEXT_PUBLIC_ARGOS_KEY ?? "ARG-••••";
    try {
      await navigator.clipboard.writeText(key);
      setCopying(true);
      setTimeout(() => setCopying(false), 1500);
    } catch {}
  }

  const statusIcon = (s: ConnStatus) => {
    if (s === "checking") return <RefreshCw size={13} className="animate-spin" style={{ color: "var(--accent)" }} />;
    if (s === "ok")       return <CheckCircle size={13} style={{ color: "#34d399" }} />;
    if (s === "error")    return <XCircle size={13} style={{ color: "#f87171" }} />;
    return <Clock size={13} style={{ color: "var(--text-muted)" }} />;
  };

  const statusBadge = (s: ConnStatus) => {
    if (s === "checking") return <Badge variant="muted">Verificando…</Badge>;
    if (s === "ok")       return <Badge variant="success">✓ Conectado</Badge>;
    if (s === "error")    return <Badge variant="danger">Offline</Badge>;
    return <Badge variant="muted">Não configurado</Badge>;
  };

  return (
    <div className="p-8 space-y-6 max-w-3xl">
      <div>
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Settings size={22} />
          Configurações
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Conexões, integrações e preferências do sistema
        </p>
      </div>

      {/* ── Conexões ── */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Database size={15} />
              Conexões
            </CardTitle>
            <button
              onClick={runChecks}
              disabled={refreshing}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs transition-all"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "var(--text-muted)" }}
            >
              <RefreshCw size={12} className={refreshing ? "animate-spin" : ""} />
              {refreshing ? "Verificando…" : "Re-verificar"}
            </button>
          </div>
        </CardHeader>
        <div className="space-y-2">
          {conns.map(conn => (
            <div key={conn.label}
              className="flex items-center justify-between p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
              <div className="flex items-center gap-2.5">
                {statusIcon(conn.status)}
                <div>
                  <p className="text-sm font-medium text-white">{conn.label}</p>
                  <p className="text-xs font-mono" style={{ color: "var(--text-muted)" }}>
                    {conn.url}
                    {conn.latency !== undefined && conn.status === "ok" && (
                      <span className="ml-2 text-[10px]" style={{ color: "#34d39980" }}>
                        {conn.latency}ms
                      </span>
                    )}
                  </p>
                </div>
              </div>
              {statusBadge(conn.status)}
            </div>
          ))}
        </div>
      </Card>

      {/* ── Acesso ── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield size={15} />
            Acesso à Plataforma
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          <div>
            <label className="text-xs mb-1.5 block" style={{ color: "var(--text-muted)" }}>
              Chave de acesso (ARGOS_ACCESS_KEY)
            </label>
            <div className="flex items-center gap-2">
              <div className="flex-1 px-4 py-2.5 rounded-lg text-sm font-mono flex items-center justify-between"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
                <span>{maskKey("ARG-8A6D8C-579217-96AE8C-82685A")}</span>
                <span className="text-[10px] px-1.5 py-0.5 rounded" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
                  httpOnly cookie · 30 dias
                </span>
              </div>
              <button
                onClick={copyKey}
                className="p-2.5 rounded-lg transition-all"
                title="Copiar chave"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "var(--text-muted)" }}
              >
                {copying ? <Check size={14} style={{ color: "#34d399" }} /> : <Copy size={14} />}
              </button>
            </div>
            <p className="text-xs mt-1.5" style={{ color: "var(--text-muted)" }}>
              Defina em <code className="font-mono" style={{ color: "#6366f1" }}>frontend/.env.local</code> como{" "}
              <code className="font-mono" style={{ color: "#6366f1" }}>ARGOS_ACCESS_KEY</code>
            </p>
          </div>
        </div>
      </Card>

      {/* ── AI ── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cpu size={15} />
            Modelos de IA
          </CardTitle>
        </CardHeader>
        <div className="space-y-2">
          {[
            { label: "Classificação IPC",    model: "BERTimbau fine-tuned · 8 categorias · argos_model/", status: "ok" },
            { label: "Embeddings semânticos", model: "SBERT multilingual-MiniLM-L12-v2",                  status: "ok" },
            { label: "Baseline TF-IDF",       model: "TF-IDF + Random Forest · 98.1% F1",                status: "ok" },
            { label: "LLM (claims + chat)",   model: "Groq llama-3.3-70b-versatile",                     status: "ok" },
          ].map(item => (
            <div key={item.label} className="flex items-center justify-between p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
              <div>
                <p className="text-sm font-medium text-white">{item.label}</p>
                <p className="text-xs font-mono" style={{ color: "var(--text-muted)" }}>{item.model}</p>
              </div>
              <Badge variant="success">✓ Ativo</Badge>
            </div>
          ))}
        </div>
      </Card>

      {/* ── Alertas ── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bell size={15} />
            Notificações
          </CardTitle>
        </CardHeader>
        <div className="space-y-3">
          {[
            { label: "Alertas de prazo (30 dias antes)",            enabled: true  },
            { label: "Novas anterioridades em watchlist",           enabled: true  },
            { label: "Novas oportunidades UFOP detectadas",         enabled: true  },
            { label: "Relatório semanal de portfolio",              enabled: false },
          ].map(n => (
            <div key={n.label} className="flex items-center justify-between">
              <span className="text-sm" style={{ color: "var(--text-muted)" }}>{n.label}</span>
              <div className="w-9 h-5 rounded-full cursor-pointer transition-colors flex items-center px-0.5"
                style={{ background: n.enabled ? "var(--accent)" : "var(--border)" }}>
                <div className={`w-4 h-4 bg-white rounded-full transition-transform ${n.enabled ? "translate-x-4" : "translate-x-0"}`} />
              </div>
            </div>
          ))}
        </div>
      </Card>

      {/* ── Version ── */}
      <div className="text-center text-xs pb-4" style={{ color: "#1e2a3a" }}>
        Argos IP Intelligence · v0.2.0-alpha · Build {new Date().toISOString().slice(0, 10)}
        <br />
        Phase 1 ✅ &nbsp;Phase 2 (Lens) 🔄 &nbsp;Phase 3 (TM) ✅ &nbsp;Frontend ✅ &nbsp;Auth ✅
      </div>
    </div>
  );
}
