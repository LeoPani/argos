"use client";

// Global toast system. Single provider mounted at root layout.
//
// Usage:
//   const toast = useToast();
//   toast.success("Contrato criado");
//   toast.error("Falha ao salvar", "Verifique a conexão");
//   toast.info("Análise IA em andamento");

import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";
import { CheckCircle2, AlertCircle, Info, X, AlertTriangle } from "lucide-react";

export type ToastKind = "success" | "error" | "info" | "warning";

interface Toast {
  id: number;
  kind: ToastKind;
  title: string;
  description?: string;
  duration: number;
}

interface ToastContextValue {
  push: (t: Omit<Toast, "id">) => void;
  success: (title: string, description?: string) => void;
  error:   (title: string, description?: string) => void;
  info:    (title: string, description?: string) => void;
  warning: (title: string, description?: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

let counter = 0;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timeoutsRef = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  const dismiss = useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id));
    const handle = timeoutsRef.current.get(id);
    if (handle) {
      clearTimeout(handle);
      timeoutsRef.current.delete(id);
    }
  }, []);

  const push = useCallback((t: Omit<Toast, "id">) => {
    const id = ++counter;
    setToasts(prev => [...prev, { id, ...t }]);
    const handle = setTimeout(() => dismiss(id), t.duration);
    timeoutsRef.current.set(id, handle);
  }, [dismiss]);

  const value: ToastContextValue = {
    push,
    success: (title, description) => push({ kind: "success", title, description, duration: 3500 }),
    error:   (title, description) => push({ kind: "error",   title, description, duration: 6000 }),
    info:    (title, description) => push({ kind: "info",    title, description, duration: 4000 }),
    warning: (title, description) => push({ kind: "warning", title, description, duration: 4500 }),
  };

  useEffect(() => {
    const t = timeoutsRef.current;
    return () => { t.forEach(h => clearTimeout(h)); };
  }, []);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
        {toasts.map(t => (
          <ToastCard key={t.id} toast={t} onClose={() => dismiss(t.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

// ─── hook ─────────────────────────────────────────────────────────────────────

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    // Fallback for use outside provider — log instead of throwing.
    return {
      push: () => {},
      success: (t) => console.log("toast.success:", t),
      error:   (t) => console.error("toast.error:", t),
      info:    (t) => console.log("toast.info:", t),
      warning: (t) => console.warn("toast.warning:", t),
    };
  }
  return ctx;
}

// ─── single card ──────────────────────────────────────────────────────────────

function ToastCard({ toast, onClose }: { toast: Toast; onClose: () => void }) {
  const style = kindStyle[toast.kind];
  const Icon  = kindIcon[toast.kind];

  return (
    <div className="toast-enter pointer-events-auto min-w-[280px] max-w-md rounded-lg p-3 flex items-start gap-3 shadow-xl"
      style={{
        background: "var(--surface)",
        border:     `1px solid ${style.border}`,
        boxShadow:  `0 10px 30px rgba(0,0,0,0.4), 0 0 0 1px ${style.border}`,
      }}>
      <div className="p-1.5 rounded-md shrink-0" style={{ background: style.iconBg }}>
        <Icon size={14} style={{ color: style.iconColor }} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-white">{toast.title}</p>
        {toast.description && (
          <p className="text-xs mt-0.5 leading-relaxed" style={{ color: "var(--text-muted)" }}>
            {toast.description}
          </p>
        )}
      </div>
      <button onClick={onClose}
        className="shrink-0 p-1 rounded hover:bg-white/10 transition-colors"
        style={{ color: "var(--text-muted)" }}
        aria-label="Fechar">
        <X size={12} />
      </button>

      <style jsx>{`
        .toast-enter {
          animation: toastIn 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        }
        @keyframes toastIn {
          from { opacity: 0; transform: translateX(20px); }
          to   { opacity: 1; transform: translateX(0); }
        }
      `}</style>
    </div>
  );
}

const kindStyle: Record<ToastKind, { border: string; iconBg: string; iconColor: string }> = {
  success: { border: "#34d39940", iconBg: "#34d39920", iconColor: "#34d399" },
  error:   { border: "#ef444440", iconBg: "#ef444420", iconColor: "#ef4444" },
  info:    { border: "#6366f140", iconBg: "#6366f120", iconColor: "#818cf8" },
  warning: { border: "#fbbf2440", iconBg: "#fbbf2420", iconColor: "#fbbf24" },
};

const kindIcon: Record<ToastKind, typeof CheckCircle2> = {
  success: CheckCircle2,
  error:   AlertCircle,
  info:    Info,
  warning: AlertTriangle,
};
