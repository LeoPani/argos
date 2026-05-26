"use client";

// AnalysisModeBadge — exibe transparentemente qual classificador está em uso.
// Honestidade científica vs. ilusão de IA.

import { useState } from "react";
import { useAnalysisMode } from "@/lib/hooks";
import { Brain, AlertCircle, Sparkles, X } from "lucide-react";

const modeStyle: Record<string, { label: string; color: string; bg: string; icon: typeof Brain }> = {
  trained_sbert: {
    label: "Modelo Sentence-BERT treinado",
    color: "#34d399",
    bg: "#34d39920",
    icon: Sparkles,
  },
  bert_fine_tuned: {
    label: "BERT fine-tuned (legado)",
    color: "#a855f7",
    bg: "#a855f720",
    icon: Brain,
  },
  heuristic: {
    label: "Heurística (não-ML)",
    color: "#fbbf24",
    bg: "#fbbf2420",
    icon: AlertCircle,
  },
};

export function AnalysisModeBadge({ compact = false }: { compact?: boolean }) {
  const { data } = useAnalysisMode();
  const [open, setOpen] = useState(false);

  if (!data) return null;
  const style = modeStyle[data.mode] ?? modeStyle.heuristic;
  const Icon = style.icon;

  return (
    <span className="relative inline-flex items-center">
      <button onClick={() => setOpen(o => !o)}
        className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs transition-all"
        style={{
          background: style.bg,
          border: `1px solid ${style.color}40`,
          color: style.color,
        }}
        title={data.description}>
        <Icon size={11} />
        {compact ? data.mode.replace("_", " ") : style.label}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-7 left-0 z-50 w-[420px] rounded-lg p-4 text-left"
            style={{
              background: "var(--surface)",
              border: "1px solid var(--border)",
              boxShadow: "0 20px 50px rgba(0,0,0,0.5)",
            }}>
            <div className="flex items-start justify-between mb-2">
              <div className="flex items-center gap-2">
                <Icon size={14} style={{ color: style.color }} />
                <span className="text-sm font-semibold" style={{ color: style.color }}>
                  {style.label}
                </span>
              </div>
              <button onClick={() => setOpen(false)} className="text-slate-500 hover:text-white">
                <X size={12} />
              </button>
            </div>

            <p className="text-xs mb-3 leading-relaxed" style={{ color: "var(--text-muted)" }}>
              {data.description}
            </p>

            {/* Status flags */}
            <div className="grid grid-cols-2 gap-2 mb-3 text-xs">
              <StatusFlag label="BERT API"   on={data.bert_online} />
              <StatusFlag label="Groq key"   on={data.groq_key_set} />
              <StatusFlag label="Claude key" on={data.anthropic_key_set} />
              <StatusFlag label="Lens token" on={data.lens_token_set} />
            </div>

            {data.annotator_provider && (
              <p className="text-xs mb-2" style={{ color: "var(--text-muted)" }}>
                Anotador ML pronto via{" "}
                <span style={{ color: "#34d399" }}>{data.annotator_provider}</span>
              </p>
            )}

            {/* Limitações */}
            {data.limitations && data.limitations.length > 0 && (
              <div className="mb-2">
                <p className="text-xs font-semibold text-white mb-1">Limitações atuais</p>
                <ul className="space-y-0.5">
                  {data.limitations.map((l, i) => (
                    <li key={i} className="text-xs leading-relaxed"
                      style={{ color: "var(--text-muted)" }}>
                      · {l}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* Próximos passos */}
            {data.next_steps && data.next_steps.length > 0 && (
              <div className="mb-2">
                <p className="text-xs font-semibold text-white mb-1">Como melhorar</p>
                <ul className="space-y-0.5">
                  {data.next_steps.map((s, i) => (
                    <li key={i} className="text-xs leading-relaxed"
                      style={{ color: "var(--text-muted)" }}>
                      → {s}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="mt-3 pt-2" style={{ borderTop: "1px solid var(--border)" }}>
              <a href="/METHODOLOGY.md" target="_blank"
                className="text-xs text-indigo-400 hover:text-indigo-300">
                METHODOLOGY.md • defesa acadêmica completa →
              </a>
            </div>
          </div>
        </>
      )}
    </span>
  );
}

function StatusFlag({ label, on }: { label: string; on: boolean }) {
  return (
    <div className="px-2 py-1 rounded text-xs flex items-center gap-1"
      style={{
        background: "var(--surface-2)",
        border: `1px solid ${on ? "#34d39940" : "#64748b40"}`,
      }}>
      <span className="w-1.5 h-1.5 rounded-full"
        style={{ background: on ? "#34d399" : "#64748b" }} />
      <span style={{ color: on ? "#34d399" : "var(--text-muted)" }}>
        {label}
      </span>
    </div>
  );
}
