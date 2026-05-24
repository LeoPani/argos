// UnderConstruction — tarja visual aplicável em features incompletas.
//
// 3 variantes:
//   <UnderConstructionBadge />  — pílula pequena (inline em headers)
//   <UnderConstructionRibbon /> — faixa diagonal no canto de um Card
//   <UnderConstructionOverlay /> — sobrepõe um placeholder bloqueante
//
// Justificativa visual: hachura diagonal âmbar/amarelo, padrão univ-
// ersalmente reconhecido de "obras". Sem texto agressivo.

import { Construction } from "lucide-react";

export function UnderConstructionBadge({ label = "em construção" }: { label?: string }) {
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
      style={{
        background: "rgba(251, 191, 36, 0.15)",
        border: "1px solid rgba(251, 191, 36, 0.35)",
        color: "#fbbf24",
      }}>
      <Construction size={10} />
      {label}
    </span>
  );
}

export function UnderConstructionRibbon() {
  return (
    <div className="absolute top-3 right-3 z-10 pointer-events-none">
      <UnderConstructionBadge />
    </div>
  );
}

export function UnderConstructionOverlay({ title, description, children }: {
  title?: string;
  description?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="relative">
      {/* Optional preview behind */}
      {children && (
        <div className="opacity-30 pointer-events-none">{children}</div>
      )}

      {/* Overlay */}
      <div className="absolute inset-0 flex items-center justify-center p-6"
        style={{
          background: "rgba(10, 10, 15, 0.85)",
          backdropFilter: "blur(2px)",
          backgroundImage: "repeating-linear-gradient(45deg, rgba(251, 191, 36, 0.05), rgba(251, 191, 36, 0.05) 10px, transparent 10px, transparent 20px)",
        }}>
        <div className="text-center max-w-md">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full mb-3"
            style={{
              background: "rgba(251, 191, 36, 0.15)",
              border: "1px solid rgba(251, 191, 36, 0.4)",
            }}>
            <Construction size={20} className="text-amber-400" />
          </div>
          <p className="text-base font-semibold text-white">
            {title ?? "Em construção"}
          </p>
          {description && (
            <p className="text-sm mt-1.5" style={{ color: "var(--text-muted)" }}>
              {description}
            </p>
          )}
          <p className="text-xs mt-2" style={{ color: "var(--text-muted)" }}>
            Veja o <a href="/roadmap" className="text-indigo-400 hover:text-indigo-300 underline">roadmap</a> para detalhes.
          </p>
        </div>
      </div>
    </div>
  );
}

// CSS for diagonal-stripe background — adicione no global se quiser usar como bg
export const constructionStripes: React.CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(45deg, rgba(251, 191, 36, 0.08), rgba(251, 191, 36, 0.08) 10px, transparent 10px, transparent 20px)",
};
