"use client";

// CitationNetworkViz — SVG force-directed graph nativo (sem libs externas).
// Radial layout: patente central no meio, citações dispostas em círculo
// (backward acima, forward abaixo). Simples mas legível.
//
// Validação metodológica:
//   Narin, F. (1994). "Patent bibliometrics." Scientometrics, 30(1), 147-155.
//   Visualização clássica de citation network.

import type { CitationNetwork } from "@/lib/types";

const groupColor: Record<string, string> = {
  self:     "#6366f1",
  forward:  "#34d399",   // quem cita NÓS — sinal de impacto
  backward: "#fbbf24",   // quem NÓS citamos — prior art
};

const ipcColors: Record<string, string> = {
  A: "#ef4444", B: "#f59e0b", C: "#34d399", D: "#06b6d4",
  E: "#3b82f6", F: "#8b5cf6", G: "#a855f7", H: "#ec4899",
};

export function CitationNetworkViz({ network }: { network: CitationNetwork }) {
  const width = 600;
  const height = 380;
  const cx = width / 2;
  const cy = height / 2;
  const radius = 130;

  const forward  = network.nodes.filter(n => n.group === "forward");
  const backward = network.nodes.filter(n => n.group === "backward");
  const self     = network.nodes.find(n => n.group === "self");

  // Place forward nodes on top semicircle, backward on bottom
  const positions = new Map<string, { x: number; y: number }>();
  if (self) positions.set(self.id, { x: cx, y: cy });

  forward.forEach((n, i) => {
    const angle = Math.PI + (i + 1) * Math.PI / (forward.length + 1);
    positions.set(n.id, {
      x: cx + radius * Math.cos(angle),
      y: cy + radius * Math.sin(angle),
    });
  });
  backward.forEach((n, i) => {
    const angle = (i + 1) * Math.PI / (backward.length + 1);
    positions.set(n.id, {
      x: cx + radius * Math.cos(angle),
      y: cy + radius * Math.sin(angle),
    });
  });

  if (network.nodes.length <= 1) {
    return (
      <div className="p-6 text-center text-sm" style={{ color: "var(--text-muted)" }}>
        Nenhuma citação registrada. Rode &quot;Enriquecer via Lens&quot; em /metricas para popular.
      </div>
    );
  }

  return (
    <div className="w-full overflow-x-auto">
      <svg width={width} height={height} className="mx-auto block">
        {/* Legend backgrounds */}
        <rect x={0} y={0} width={width} height={cy}
          fill="rgba(52, 211, 153, 0.03)" />
        <rect x={0} y={cy} width={width} height={cy}
          fill="rgba(251, 191, 36, 0.03)" />
        <text x={width - 8} y={20} textAnchor="end"
          fontSize={10} fill="#34d399">↑ FORWARD (impacto)</text>
        <text x={width - 8} y={height - 8} textAnchor="end"
          fontSize={10} fill="#fbbf24">↓ BACKWARD (prior art)</text>

        {/* Links */}
        {network.links.map((link, i) => {
          const s = positions.get(link.source);
          const t = positions.get(link.target);
          if (!s || !t) return null;
          const color = link.kind === "forward" ? "#34d399" : "#fbbf24";
          return (
            <line key={i} x1={s.x} y1={s.y} x2={t.x} y2={t.y}
              stroke={color} strokeOpacity={0.4} strokeWidth={1.5} />
          );
        })}

        {/* Nodes */}
        {network.nodes.map(n => {
          const p = positions.get(n.id);
          if (!p) return null;
          const isSelf = n.group === "self";
          const r = isSelf ? 22 : 14;
          const color = n.ipc && ipcColors[n.ipc] ? ipcColors[n.ipc] : groupColor[n.group];

          return (
            <g key={n.id}>
              {/* Halo for self */}
              {isSelf && (
                <circle cx={p.x} cy={p.y} r={r + 6}
                  fill={color} fillOpacity={0.15} />
              )}
              <circle cx={p.x} cy={p.y} r={r}
                fill={color}
                stroke="#0a0a0f" strokeWidth={2} />
              <text x={p.x} y={p.y + 4} textAnchor="middle"
                fontSize={isSelf ? 12 : 9}
                fontWeight={isSelf ? "bold" : "normal"}
                fill="white">
                {n.ipc ?? (isSelf ? "📜" : "·")}
              </text>
              {/* Year label below */}
              {n.year && (
                <text x={p.x} y={p.y + r + 12} textAnchor="middle"
                  fontSize={9} fill="var(--text-muted)">
                  {n.year}
                </text>
              )}
            </g>
          );
        })}
      </svg>

      {/* Stats footer */}
      <div className="flex justify-center gap-4 mt-3 text-xs">
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full" style={{ background: "#6366f1" }} />
          <span style={{ color: "var(--text-muted)" }}>Esta patente</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full" style={{ background: "#34d399" }} />
          <span style={{ color: "var(--text-muted)" }}>{network.stats.forward_count} forward (impacto)</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full" style={{ background: "#fbbf24" }} />
          <span style={{ color: "var(--text-muted)" }}>{network.stats.backward_count} backward (prior art)</span>
        </div>
        {network.stats.avg_year > 0 && (
          <span style={{ color: "var(--text-muted)" }}>
            · Ano médio: <span className="text-white">{network.stats.avg_year.toFixed(0)}</span>
          </span>
        )}
      </div>
    </div>
  );
}
