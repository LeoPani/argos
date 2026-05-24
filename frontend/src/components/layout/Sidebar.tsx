"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Search,
  FolderOpen,
  Scale,
  GraduationCap,
  Landmark,
  Bell,
  MessageSquare,
  Settings,
  Award,
  Sparkles,
  Briefcase,
  Calendar as CalendarIcon,
} from "lucide-react";

const nav = [
  { href: "/dashboard", icon: LayoutDashboard, label: "BI & Analytics" },
  { href: "/metricas",  icon: Award, label: "Métricas Acadêmicas" },
  { href: "/smart-filing", icon: Sparkles, label: "Smart Filing" },
  { href: "/marketplace",  icon: Briefcase,    label: "TT Marketplace" },
  { href: "/calendario",   icon: CalendarIcon, label: "Calendário NIT" },
  { href: "/consultas", icon: Search, label: "Consultas" },
  { href: "/portfolio", icon: FolderOpen, label: "Portfolio de PI" },
  { href: "/arbitragem", icon: Scale, label: "Arbitragem" },
  { href: "/ufop", icon: GraduationCap, label: "UFOP Intelligence" },
  { href: "/pool", icon: Landmark, label: "Pool & TT" },
  { href: "/alertas", icon: Bell, label: "Alertas" },
  { href: "/chat", icon: MessageSquare, label: "Chat de PI" },
];

export function Sidebar() {
  const path = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 w-56 flex flex-col z-40"
      style={{ background: "var(--surface)", borderRight: "1px solid var(--border)" }}>

      {/* Logo */}
      <div className="flex items-center gap-3 px-5 py-5"
        style={{ borderBottom: "1px solid var(--border)" }}>
        <EyeLogo />
        <div>
          <p className="text-sm font-semibold tracking-widest text-white">ARGOS</p>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>IP Intelligence</p>
        </div>
      </div>

      {/* ⌘K hint */}
      <button
        onClick={() => {
          window.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true }));
        }}
        className="mx-2 mt-3 flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-all"
        style={{
          background: "var(--surface-2)", border: "1px solid var(--border)",
          color: "var(--text-muted)",
        }}>
        <Search size={13} />
        <span className="flex-1 text-left text-xs">Buscar tudo…</span>
        <kbd className="font-mono text-[10px] px-1.5 py-0.5 rounded"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          ⌘K
        </kbd>
      </button>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-4 px-2">
        {nav.map(({ href, icon: Icon, label }) => {
          const active = path.startsWith(href);
          return (
            <Link key={href} href={href}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg mb-0.5 text-sm transition-all",
                active
                  ? "text-white font-medium"
                  : "hover:text-white"
              )}
              style={{
                color: active ? "white" : "var(--text-muted)",
                background: active ? "var(--accent)" : "transparent",
              }}
              onMouseEnter={e => { if (!active) (e.currentTarget as HTMLElement).style.background = "var(--surface-2)"; }}
              onMouseLeave={e => { if (!active) (e.currentTarget as HTMLElement).style.background = "transparent"; }}
            >
              <Icon size={16} />
              {label}
            </Link>
          );
        })}
      </nav>

      {/* Bottom */}
      <div className="px-2 py-3" style={{ borderTop: "1px solid var(--border)" }}>
        <Link href="/config"
          className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all"
          style={{ color: "var(--text-muted)" }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = "var(--surface-2)"; }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = "transparent"; }}
        >
          <Settings size={16} />
          Configurações
        </Link>
      </div>
    </aside>
  );
}

// ArgosPanoptesLogo — homenagem ao gigante de muitos olhos da mitologia grega.
// Olho central dominante + constelação de pequenos olhos em arranjo hexagonal.
// Sem agressividade: pupilas circulares, sem dentes/garras, paleta indigo
// contemplativa. Inspiração Lilith mas pacificada.
function EyeLogo() {
  return (
    <svg width="36" height="36" viewBox="0 0 36 36" fill="none" aria-label="Argos">
      {/* Halo externo */}
      <circle cx="18" cy="18" r="17" stroke="#6366f1" strokeWidth="0.8" strokeOpacity="0.3" />

      {/* Olho central (Argos) */}
      <ellipse cx="18" cy="18" rx="9" ry="6.5" stroke="#818cf8" strokeWidth="1.4" fill="rgba(99,102,241,0.08)" />
      <circle cx="18" cy="18" r="3.5" fill="#6366f1" />
      <circle cx="18.8" cy="17.2" r="0.8" fill="white" opacity="0.7" />

      {/* Olhos secundários em arranjo hexagonal (6 olhos = simbólico) */}
      {/* Topo */}
      <circle cx="18" cy="4.5" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="18" cy="4.5" r="0.4" fill="white" opacity="0.6" />
      {/* Topo-direita */}
      <circle cx="30" cy="11" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="30" cy="11" r="0.4" fill="white" opacity="0.6" />
      {/* Bottom-direita */}
      <circle cx="30" cy="25" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="30" cy="25" r="0.4" fill="white" opacity="0.6" />
      {/* Bottom */}
      <circle cx="18" cy="31.5" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="18" cy="31.5" r="0.4" fill="white" opacity="0.6" />
      {/* Bottom-esquerda */}
      <circle cx="6" cy="25" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="6" cy="25" r="0.4" fill="white" opacity="0.6" />
      {/* Topo-esquerda */}
      <circle cx="6" cy="11" r="1.4" fill="#6366f1" opacity="0.55" />
      <circle cx="6" cy="11" r="0.4" fill="white" opacity="0.6" />

      {/* Linhas sutis ligando olho central aos secundários (network of vigilance) */}
      <g stroke="#6366f1" strokeOpacity="0.18" strokeWidth="0.4">
        <line x1="18" y1="11.5" x2="18" y2="5.5" />
        <line x1="25" y1="14"   x2="29" y2="11.5" />
        <line x1="25" y1="22"   x2="29" y2="24.5" />
        <line x1="18" y1="24.5" x2="18" y2="30.5" />
        <line x1="11" y1="22"   x2="7"  y2="24.5" />
        <line x1="11" y1="14"   x2="7"  y2="11.5" />
      </g>
    </svg>
  );
}
