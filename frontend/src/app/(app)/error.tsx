"use client";

import { useEffect } from "react";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("[Argos] page error:", error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-screen gap-0"
      style={{ background: "var(--bg)" }}>

      {/* Argos eye — cracked/error */}
      <svg width="100" height="68" viewBox="0 0 120 80" fill="none" style={{ marginBottom: 28 }}>
        <defs>
          <radialGradient id="irisErr" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#ef4444" stopOpacity="0.8" />
            <stop offset="100%" stopColor="#ef4444" stopOpacity="0.1" />
          </radialGradient>
        </defs>
        <path d="M8 40 C20 15 40 6 60 6 C80 6 100 15 112 40 C100 65 80 74 60 74 C40 74 20 65 8 40Z"
          stroke="#ef444440" strokeWidth="1.5" fill="#ef444408" />
        <circle cx="60" cy="40" r="22" fill="url(#irisErr)" />
        <circle cx="60" cy="40" r="22" stroke="#ef4444" strokeWidth="1" fill="none" opacity="0.6" />
        <circle cx="60" cy="40" r="11" fill="#07070e" />
        <circle cx="60" cy="40" r="3" fill="#ef4444" opacity="0.9" />
        {/* crack lines */}
        <path d="M52 28 L58 40 L50 52" stroke="#ef444460" strokeWidth="0.8" fill="none" strokeLinecap="round" />
        <path d="M68 26 L62 40 L70 50" stroke="#ef444460" strokeWidth="0.8" fill="none" strokeLinecap="round" />
      </svg>

      <h1 className="text-4xl font-bold" style={{ color: "#ef4444", letterSpacing: "0.06em", margin: 0 }}>
        500
      </h1>
      <p className="text-sm uppercase tracking-widest mt-2" style={{ color: "#7f1d1d" }}>
        Erro interno
      </p>
      <p className="text-xs mt-2 text-center max-w-xs" style={{ color: "#374151" }}>
        {error.message || "Algo inesperado aconteceu. O erro foi registrado."}
      </p>
      {error.digest && (
        <p className="text-xs mt-1 font-mono" style={{ color: "#1f2937" }}>
          ref: {error.digest}
        </p>
      )}

      <button
        onClick={reset}
        className="mt-9 px-7 py-2.5 rounded-lg text-sm font-semibold tracking-wide transition-colors"
        style={{ background: "#ef4444", color: "#fff" }}
        onMouseEnter={e => (e.currentTarget.style.background = "#f87171")}
        onMouseLeave={e => (e.currentTarget.style.background = "#ef4444")}
      >
        Tentar novamente
      </button>
    </div>
  );
}
