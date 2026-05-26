"use client";

// DataStateBanner — strip global no topo do dashboard com o "estado dos dados".
// Mostra honestamente o que é real, o que é heurístico e o que está stub.
// Dispensável (localStorage flag) — para não poluir uso diário.

import { useEffect, useState } from "react";
import { useAnalysisMode } from "@/lib/hooks";
import { ShieldCheck, AlertTriangle, X } from "lucide-react";

const DISMISS_KEY = "argos.dataStateBanner.dismissed";

export function DataStateBanner() {
  const { data } = useAnalysisMode();
  const [dismissed, setDismissed] = useState<boolean>(true);

  useEffect(() => {
    if (typeof window === "undefined") return;
    setDismissed(localStorage.getItem(DISMISS_KEY) === "1");
  }, []);

  if (!data || dismissed) return null;

  const onDismiss = () => {
    setDismissed(true);
    if (typeof window !== "undefined") localStorage.setItem(DISMISS_KEY, "1");
  };

  const sources = Object.entries(data.data_sources || {});
  const isHonestlyDegraded = data.mode === "heuristic" || (data.limitations?.length ?? 0) > 0;
  const accent = isHonestlyDegraded ? "#fbbf24" : "#34d399";
  const Icon = isHonestlyDegraded ? AlertTriangle : ShieldCheck;

  return (
    <div
      className="rounded-lg p-3 flex items-start gap-3"
      style={{
        background: `${accent}10`,
        border: `1px solid ${accent}40`,
      }}
    >
      <Icon size={16} style={{ color: accent, flexShrink: 0, marginTop: 2 }} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-semibold" style={{ color: accent }}>
            Estado dos dados
          </span>
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            · transparência acadêmica
          </span>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-1">
          {sources.map(([key, value]) => (
            <div key={key} className="text-xs leading-relaxed">
              <span className="font-medium text-white">{labelize(key)}:</span>{" "}
              <span style={{ color: "var(--text-muted)" }}>{value}</span>
            </div>
          ))}
        </div>
        {data.mode === "heuristic" && (
          <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
            Classificador atual: <strong style={{ color: accent }}>heurística</strong>{" "}
            (não-ML). Pipeline de treino disponível em{" "}
            <code className="text-[11px] px-1 py-0.5 rounded"
              style={{ background: "var(--surface-2)" }}>
              ai-service/training/
            </code>{" "}
            — precisa de{" "}
            <code className="text-[11px] px-1 py-0.5 rounded"
              style={{ background: "var(--surface-2)" }}>
              GROQ_API_KEY
            </code>{" "}
            (free 14400/dia) ou{" "}
            <code className="text-[11px] px-1 py-0.5 rounded"
              style={{ background: "var(--surface-2)" }}>
              ANTHROPIC_API_KEY
            </code>.
          </p>
        )}
      </div>
      <button
        onClick={onDismiss}
        className="text-slate-500 hover:text-white flex-shrink-0"
        title="Fechar (você pode reabrir limpando localStorage)"
      >
        <X size={14} />
      </button>
    </div>
  );
}

function labelize(key: string): string {
  return key
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}
