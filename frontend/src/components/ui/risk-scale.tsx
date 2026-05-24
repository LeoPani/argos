"use client";

import { riskBg, riskColor, riskLabel } from "@/lib/utils";
import { cn } from "@/lib/utils";

interface RiskScaleProps {
  score: number; // 0–10
  showLabel?: boolean;
  size?: "sm" | "md" | "lg";
}

export function RiskScale({ score, showLabel = true, size = "md" }: RiskScaleProps) {
  const pct = (score / 10) * 100;
  const color = riskBg(score);
  const textColor = riskColor(score);
  const label = riskLabel(score);

  const heights = { sm: "h-1.5", md: "h-2.5", lg: "h-3.5" };
  const textSizes = { sm: "text-xs", md: "text-sm", lg: "text-base" };

  return (
    <div className="w-full">
      {showLabel && (
        <div className="flex items-center justify-between mb-1.5">
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>
            Escala de risco
          </span>
          <span className={cn("font-bold", textSizes[size], textColor)}>
            {score.toFixed(1)} / 10 — {label}
          </span>
        </div>
      )}
      <div
        className={cn("w-full rounded-full overflow-hidden", heights[size])}
        style={{ background: "var(--border)" }}
      >
        <div
          className={cn("h-full rounded-full transition-all duration-700", color)}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && (
        <div className="flex justify-between mt-1">
          <span className="text-xs text-emerald-400">Baixo</span>
          <span className="text-xs text-amber-400">Médio</span>
          <span className="text-xs text-orange-400">Alto</span>
          <span className="text-xs text-red-400">Muito Alto</span>
        </div>
      )}
    </div>
  );
}
