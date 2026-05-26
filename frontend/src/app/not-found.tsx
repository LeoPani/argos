"use client";

import Link from "next/link";

export default function NotFound() {
  return (
    <div style={{
      position: "fixed", inset: 0,
      display: "flex", alignItems: "center", justifyContent: "center",
      background: "#07070e", flexDirection: "column", gap: 0,
    }}>
      {/* Argos eye — sad/closed */}
      <svg width="100" height="68" viewBox="0 0 120 80" fill="none" style={{ marginBottom: 28, opacity: 0.5 }}>
        <defs>
          <radialGradient id="iris404" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#6366f1" stopOpacity="0.6" />
            <stop offset="100%" stopColor="#6366f1" stopOpacity="0.1" />
          </radialGradient>
        </defs>
        {/* Closed eye — just a line */}
        <path d="M8 40 C20 15 40 6 60 6 C80 6 100 15 112 40" stroke="#6366f140" strokeWidth="1.5" fill="none" />
        {/* Closed lid line */}
        <path d="M15 40 C30 36 50 34 60 34 C70 34 90 36 105 40" stroke="#6366f1" strokeWidth="1.8" strokeLinecap="round" fill="none" />
        {/* Lashes */}
        {[20, 35, 50, 65, 80, 95].map((x, i) => (
          <line key={i} x1={x} y1={37} x2={x - 2 + i} y2={30} stroke="#6366f160" strokeWidth="0.8" strokeLinecap="round" />
        ))}
      </svg>

      <h1 style={{ fontSize: "5rem", fontWeight: 800, color: "#1e2a4a", letterSpacing: "0.1em", margin: 0 }}>
        404
      </h1>
      <p style={{ fontSize: "0.9rem", color: "#334155", letterSpacing: "0.12em", textTransform: "uppercase", margin: "8px 0 0" }}>
        Página não encontrada
      </p>
      <p style={{ fontSize: "0.78rem", color: "#1e293b", marginTop: 8, textAlign: "center", maxWidth: 320 }}>
        O Argos vasculhou todos os registros mas não encontrou esta rota.
      </p>

      <Link href="/dashboard" style={{
        marginTop: 36,
        padding: "11px 28px",
        background: "#6366f1",
        color: "#fff",
        borderRadius: 10,
        fontSize: "0.88rem",
        fontWeight: 600,
        letterSpacing: "0.06em",
        textDecoration: "none",
        transition: "background 0.15s",
      }}
        onMouseEnter={e => (e.currentTarget.style.background = "#818cf8")}
        onMouseLeave={e => (e.currentTarget.style.background = "#6366f1")}
      >
        Voltar ao Dashboard
      </Link>
    </div>
  );
}
