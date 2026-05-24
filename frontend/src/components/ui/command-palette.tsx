"use client";

// Global ⌘K / Ctrl+K palette. Federated search across patents,
// trademarks, disputes and contracts. Keyboard nav with ↑↓ + Enter.
// Mounted once at root layout — listens globally.

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { SearchHit } from "@/lib/types";
import {
  Search, FileText, Tag as TagIcon, Scale, Briefcase,
  CornerDownLeft, Command,
} from "lucide-react";

const kindMeta: Record<SearchHit["kind"], { icon: typeof Search; color: string; label: string }> = {
  patent:    { icon: FileText,  color: "#6366f1", label: "Patente"   },
  trademark: { icon: TagIcon,   color: "#f59e0b", label: "Marca"     },
  dispute:   { icon: Scale,     color: "#ef4444", label: "Disputa"   },
  contract:  { icon: Briefcase, color: "#34d399", label: "Contrato"  },
};

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen]       = useState(false);
  const [q, setQ]             = useState("");
  const [hits, setHits]       = useState<SearchHit[]>([]);
  const [loading, setLoading] = useState(false);
  const [active, setActive]   = useState(0);
  const inputRef              = useRef<HTMLInputElement>(null);
  const debounceRef           = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ⌘K / Ctrl+K to toggle
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen(o => !o);
      } else if (e.key === "Escape" && open) {
        setOpen(false);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open]);

  // Focus input on open
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
      setActive(0);
    } else {
      setQ(""); setHits([]);
    }
  }, [open]);

  // Debounced search
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!q.trim()) { setHits([]); setLoading(false); return; }
    setLoading(true);
    debounceRef.current = setTimeout(async () => {
      try {
        const r = await api.search.query(q, 5);
        setHits(r.hits);
        setActive(0);
      } catch { setHits([]); }
      finally   { setLoading(false); }
    }, 180);
  }, [q]);

  function handleSelect(hit: SearchHit) {
    router.push(hit.url);
    setOpen(false);
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === "ArrowDown") { e.preventDefault(); setActive(a => Math.min(a + 1, hits.length - 1)); }
    if (e.key === "ArrowUp")   { e.preventDefault(); setActive(a => Math.max(a - 1, 0)); }
    if (e.key === "Enter" && hits[active]) { e.preventDefault(); handleSelect(hits[active]); }
  }

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-32 px-4 palette-backdrop"
      onClick={() => setOpen(false)}>
      <div className="w-full max-w-2xl rounded-xl overflow-hidden palette-modal"
        onClick={e => e.stopPropagation()}
        style={{
          background: "var(--surface)",
          border: "1px solid var(--border)",
          boxShadow: "0 25px 60px rgba(0,0,0,0.6)",
        }}>
        {/* Input */}
        <div className="flex items-center gap-3 px-4 py-3"
          style={{ borderBottom: "1px solid var(--border)" }}>
          <Search size={16} className="text-slate-500 shrink-0" />
          <input ref={inputRef}
            value={q}
            onChange={e => setQ(e.target.value)}
            onKeyDown={handleKey}
            placeholder="Buscar patentes, marcas, disputas, contratos…"
            className="flex-1 bg-transparent outline-none text-sm text-white"
          />
          {loading && (
            <div className="w-3 h-3 border-2 border-slate-600 border-t-indigo-400 rounded-full animate-spin shrink-0" />
          )}
          <kbd className="text-xs px-1.5 py-0.5 rounded font-mono shrink-0"
            style={{ background: "var(--surface-2)", color: "var(--text-muted)", border: "1px solid var(--border)" }}>
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div className="max-h-[60vh] overflow-y-auto">
          {q.trim() === "" && (
            <div className="py-8 text-center">
              <Command size={20} className="mx-auto mb-2 text-slate-600" />
              <p className="text-xs" style={{ color: "var(--text-muted)" }}>
                Busca federada em todo o banco — comece a digitar
              </p>
              <div className="mt-3 flex justify-center gap-2 text-xs" style={{ color: "var(--text-muted)" }}>
                <Hint icon={FileText}  label="Patentes" />
                <Hint icon={TagIcon}   label="Marcas" />
                <Hint icon={Scale}     label="Disputas" />
                <Hint icon={Briefcase} label="Contratos" />
              </div>
            </div>
          )}

          {q.trim() !== "" && hits.length === 0 && !loading && (
            <p className="py-8 text-center text-sm" style={{ color: "var(--text-muted)" }}>
              Nenhum resultado para <span className="font-mono text-white">"{q}"</span>
            </p>
          )}

          {hits.map((h, i) => {
            const meta = kindMeta[h.kind];
            const Icon = meta.icon;
            const isActive = i === active;
            return (
              <button key={`${h.kind}-${h.id}`}
                onClick={() => handleSelect(h)}
                onMouseEnter={() => setActive(i)}
                className="w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors"
                style={{ background: isActive ? "var(--surface-2)" : "transparent" }}>
                <div className="p-1.5 rounded-md shrink-0" style={{ background: meta.color + "20" }}>
                  <Icon size={12} style={{ color: meta.color }} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-xs uppercase font-medium shrink-0"
                      style={{ color: meta.color }}>{meta.label}</span>
                    <span className="font-mono text-xs text-indigo-400 shrink-0">{h.reference}</span>
                  </div>
                  <p className="text-sm text-white truncate">{h.title}</p>
                  {h.subtitle && (
                    <p className="text-xs truncate" style={{ color: "var(--text-muted)" }}>{h.subtitle}</p>
                  )}
                </div>
                {isActive && (
                  <CornerDownLeft size={12} className="shrink-0" style={{ color: "var(--text-muted)" }} />
                )}
              </button>
            );
          })}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-2 text-xs"
          style={{ borderTop: "1px solid var(--border)", background: "var(--surface-2)", color: "var(--text-muted)" }}>
          <span>{hits.length > 0 ? `${hits.length} resultados` : ""}</span>
          <div className="flex gap-3">
            <kbd className="px-1 py-0.5 rounded text-[10px]" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>↑↓</kbd> navegar
            <kbd className="px-1 py-0.5 rounded text-[10px]" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>↵</kbd> abrir
          </div>
        </div>
      </div>

      <style jsx>{`
        .palette-backdrop {
          background: rgba(10, 10, 15, 0.7);
          backdrop-filter: blur(4px);
          animation: bdFadeIn 0.15s ease-out;
        }
        .palette-modal { animation: modalIn 0.18s cubic-bezier(0.16, 1, 0.3, 1); }
        @keyframes bdFadeIn { from { opacity: 0; } to { opacity: 1; } }
        @keyframes modalIn  { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </div>
  );
}

function Hint({ icon: Icon, label }: { icon: typeof Search; label: string }) {
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded"
      style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
      <Icon size={10} /> {label}
    </span>
  );
}
