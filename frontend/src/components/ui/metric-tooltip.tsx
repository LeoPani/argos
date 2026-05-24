"use client";

// MetricTooltip — small ⓘ icon that opens a popover with the methodology
// description + formula + references for a given metric. Pulls from
// the /api/v1/metrics/methodology endpoint and caches.

import { useState } from "react";
import { Info, ExternalLink } from "lucide-react";
import { useMethodology } from "@/lib/hooks";

interface MetricTooltipProps {
  metricID: string;       // "autm_health_score" | "tt_funnel" | "hjt_diversity" | etc
}

export function MetricTooltip({ metricID }: MetricTooltipProps) {
  const { data } = useMethodology();
  const [open, setOpen] = useState(false);

  const meta = data?.metrics.find(m => m.id === metricID);

  return (
    <span className="relative inline-flex items-center">
      <button onClick={() => setOpen(o => !o)}
        className="text-slate-500 hover:text-indigo-400 transition-colors p-0.5 rounded"
        aria-label="Sobre a métrica">
        <Info size={11} />
      </button>

      {open && meta && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute left-0 top-5 z-50 w-80 rounded-lg p-3 text-left"
            style={{
              background: "var(--surface)",
              border: "1px solid var(--border)",
              boxShadow: "0 10px 30px rgba(0,0,0,0.4)",
            }}>
            <p className="text-sm font-semibold text-white">{meta.name}</p>
            <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
              {meta.description}
            </p>
            <div className="mt-2 p-2 rounded font-mono text-xs"
              style={{ background: "var(--surface-2)", color: "#a5b4fc" }}>
              {meta.formula}
            </div>
            {meta.interpretation && (
              <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
                <span className="text-white font-medium">Interpretação:</span> {meta.interpretation}
              </p>
            )}
            {meta.data_requirements && (
              <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
                <span className="text-white font-medium">Requer:</span> {meta.data_requirements}
              </p>
            )}
            <div className="mt-2 pt-2" style={{ borderTop: "1px solid var(--border)" }}>
              <p className="text-xs font-medium text-white mb-1">Referências</p>
              {meta.references.map((ref, i) => (
                <p key={i} className="text-xs leading-relaxed mb-1"
                  style={{ color: "var(--text-muted)" }}>
                  • {ref}
                </p>
              ))}
            </div>
            <a href="/metodologia" className="mt-2 inline-flex items-center gap-1 text-xs text-indigo-400 hover:text-indigo-300">
              <ExternalLink size={9} /> Ver metodologia completa
            </a>
          </div>
        </>
      )}
    </span>
  );
}
