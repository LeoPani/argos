"use client";

import { useState, useEffect, useRef, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";

export default function LoginPage() {
  return (
    <Suspense>
      <LoginContent />
    </Suspense>
  );
}

function LoginContent() {
  const router      = useRouter();
  const params      = useSearchParams();
  const inputRef    = useRef<HTMLInputElement>(null);

  const [key, setKey]       = useState("");
  const [loading, setLoad]  = useState(false);
  const [error, setError]   = useState("");
  const [ok, setOk]         = useState(false);

  useEffect(() => { inputRef.current?.focus(); }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!key.trim() || loading) return;
    setLoad(true);
    setError("");

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: key.trim() }),
      });

      if (res.ok) {
        setOk(true);
        // Redirecionar após animação breve
        setTimeout(() => {
          const from = params.get("from") ?? "/dashboard";
          router.push(from);
          router.refresh();
        }, 800);
      } else {
        const data = await res.json();
        setError(data.error ?? "Chave inválida.");
        setKey("");
        inputRef.current?.focus();
      }
    } catch {
      setError("Erro de rede. Tente novamente.");
    } finally {
      setLoad(false);
    }
  }

  return (
    <div className="login-root">
      {/* Fundo com partículas sutis */}
      <div className="login-bg" aria-hidden />

      <div className="login-card">
        {/* ── Símbolo Argos ── */}
        <div className={`argos-eye ${ok ? "eye-success" : ""}`}>
          <ArgosEye authenticated={ok} />
        </div>

        {/* ── Identidade ── */}
        <h1 className="login-title">ARGOS</h1>
        <p className="login-sub">
          Inteligência Competitiva em Propriedade Intelectual
        </p>

        {/* ── Formulário ── */}
        <form onSubmit={handleSubmit} className="login-form">
          <div className={`login-field ${error ? "field-error" : ""} ${ok ? "field-ok" : ""}`}>
            <input
              ref={inputRef}
              type="password"
              value={key}
              onChange={(e) => { setKey(e.target.value); setError(""); }}
              placeholder="Chave de acesso"
              disabled={loading || ok}
              autoComplete="current-password"
              spellCheck={false}
            />
          </div>

          {error && (
            <p className="login-error">{error}</p>
          )}

          <button
            type="submit"
            disabled={!key.trim() || loading || ok}
            className="login-btn"
          >
            {ok ? (
              <span className="btn-ok">✓ Acesso concedido</span>
            ) : loading ? (
              <span className="btn-loading">
                <span className="spinner" />
                Verificando…
              </span>
            ) : (
              "Acessar"
            )}
          </button>
        </form>

        <p className="login-footer">
          UFOP · NIT · {new Date().getFullYear()}
        </p>
      </div>

      <style jsx global>{`
        /* ── Reset para a página de login ── */
        body { overflow: hidden; }

        .login-root {
          position: fixed; inset: 0;
          display: flex; align-items: center; justify-content: center;
          background: #07070e;
          z-index: 9999;
        }

        /* Grade de pontos ultra-sutil */
        .login-bg {
          position: absolute; inset: 0; pointer-events: none;
          background-image:
            radial-gradient(circle at 50% 50%, rgba(99,102,241,0.06) 0%, transparent 65%),
            radial-gradient(#1e1e30 1px, transparent 1px);
          background-size: 100% 100%, 32px 32px;
        }

        /* ── Card central ── */
        .login-card {
          position: relative; z-index: 1;
          display: flex; flex-direction: column; align-items: center;
          gap: 0;
          animation: fadeUp 0.6s ease both;
        }

        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(20px); }
          to   { opacity: 1; transform: translateY(0); }
        }

        /* ── Olho Argos ── */
        .argos-eye {
          margin-bottom: 28px;
          filter: drop-shadow(0 0 32px rgba(99,102,241,0.35));
          transition: filter 0.4s ease;
        }
        .argos-eye.eye-success {
          filter: drop-shadow(0 0 48px rgba(52,211,153,0.6));
        }

        /* ── Títulos ── */
        .login-title {
          font-size: 2.25rem;
          font-weight: 800;
          letter-spacing: 0.35em;
          color: #fff;
          margin: 0 0 8px;
          text-shadow: 0 0 40px rgba(99,102,241,0.4);
        }

        .login-sub {
          font-size: 0.8rem;
          color: #475569;
          letter-spacing: 0.08em;
          text-align: center;
          margin: 0 0 40px;
          text-transform: uppercase;
        }

        /* ── Formulário ── */
        .login-form {
          display: flex; flex-direction: column; align-items: center;
          gap: 12px; width: 320px;
        }

        .login-field {
          width: 100%;
          border-radius: 10px;
          border: 1px solid #2a2a3a;
          background: #0f0f1a;
          transition: border-color 0.2s, box-shadow 0.2s;
        }
        .login-field:focus-within {
          border-color: #6366f1;
          box-shadow: 0 0 0 3px rgba(99,102,241,0.15);
        }
        .login-field.field-error {
          border-color: #ef4444;
          box-shadow: 0 0 0 3px rgba(239,68,68,0.12);
          animation: shake 0.4s ease;
        }
        .login-field.field-ok {
          border-color: #34d399;
          box-shadow: 0 0 0 3px rgba(52,211,153,0.15);
        }

        @keyframes shake {
          0%,100% { transform: translateX(0); }
          20%,60%  { transform: translateX(-6px); }
          40%,80%  { transform: translateX(6px); }
        }

        .login-field input {
          width: 100%; padding: 14px 18px;
          background: transparent; border: none; outline: none;
          color: #e2e8f0; font-size: 0.95rem;
          letter-spacing: 0.12em; text-align: center;
          font-family: "SF Mono", "Fira Code", monospace;
        }
        .login-field input::placeholder { color: #334155; letter-spacing: 0.04em; font-family: system-ui; }

        .login-error {
          font-size: 0.78rem; color: #f87171;
          text-align: center; margin: 0;
          animation: fadeIn 0.2s ease;
        }
        @keyframes fadeIn {
          from { opacity: 0; transform: translateY(-4px); }
          to   { opacity: 1; transform: translateY(0); }
        }

        .login-btn {
          width: 100%; padding: 13px;
          background: #6366f1; color: #fff;
          border: none; border-radius: 10px;
          font-size: 0.9rem; font-weight: 600;
          letter-spacing: 0.06em; cursor: pointer;
          transition: background 0.15s, transform 0.1s, box-shadow 0.15s;
        }
        .login-btn:hover:not(:disabled) {
          background: #818cf8;
          box-shadow: 0 4px 20px rgba(99,102,241,0.4);
          transform: translateY(-1px);
        }
        .login-btn:active:not(:disabled) { transform: translateY(0); }
        .login-btn:disabled { opacity: 0.5; cursor: not-allowed; }

        .btn-loading {
          display: flex; align-items: center; justify-content: center; gap: 8px;
        }
        .btn-ok { color: #fff; }

        .spinner {
          width: 14px; height: 14px;
          border: 2px solid rgba(255,255,255,0.3);
          border-top-color: #fff;
          border-radius: 50%;
          animation: spin 0.7s linear infinite;
          display: inline-block;
        }
        @keyframes spin { to { transform: rotate(360deg); } }

        .login-footer {
          margin-top: 32px;
          font-size: 0.7rem; color: #1e2a3a;
          letter-spacing: 0.1em; text-transform: uppercase;
        }
      `}</style>
    </div>
  );
}

// ── Símbolo SVG do Argos ────────────────────────────────────────────────────

function ArgosEye({ authenticated }: { authenticated: boolean }) {
  const accent   = authenticated ? "#34d399" : "#6366f1";
  const accentSoft = authenticated ? "#34d39930" : "#6366f130";
  const accentMid  = authenticated ? "#34d39960" : "#6366f160";

  return (
    <svg width="120" height="80" viewBox="0 0 120 80" fill="none" xmlns="http://www.w3.org/2000/svg">
      {/* Definições — gradientes e filtros */}
      <defs>
        <radialGradient id="irisGrad" cx="50%" cy="50%" r="50%">
          <stop offset="0%"   stopColor={accent} stopOpacity="0.9" />
          <stop offset="60%"  stopColor={accent} stopOpacity="0.5" />
          <stop offset="100%" stopColor={accent} stopOpacity="0.1" />
        </radialGradient>
        <radialGradient id="pupilGrad" cx="40%" cy="35%" r="60%">
          <stop offset="0%"  stopColor="#1a1a2e" />
          <stop offset="100%" stopColor="#07070e" />
        </radialGradient>
        <filter id="glow">
          <feGaussianBlur stdDeviation="3" result="blur" />
          <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
      </defs>

      {/* Raios externos — sutis */}
      {[0, 30, 60, 90, 120, 150, 210, 240, 270, 300, 330].map((angle, i) => {
        const rad = (angle * Math.PI) / 180;
        const x1 = 60 + Math.cos(rad) * 46;
        const y1 = 40 + Math.sin(rad) * 30;
        const x2 = 60 + Math.cos(rad) * 58;
        const y2 = 40 + Math.sin(rad) * 38;
        return (
          <line key={i} x1={x1} y1={y1} x2={x2} y2={y2}
            stroke={accentMid} strokeWidth="0.8" strokeLinecap="round"
            opacity="0.6" />
        );
      })}

      {/* Contorno do olho */}
      <path
        d="M8 40 C20 15 40 6 60 6 C80 6 100 15 112 40 C100 65 80 74 60 74 C40 74 20 65 8 40Z"
        stroke={accent} strokeWidth="1.5" fill={accentSoft}
        filter="url(#glow)"
      />

      {/* Íris */}
      <circle cx="60" cy="40" r="22" fill="url(#irisGrad)" />

      {/* Anel da íris */}
      <circle cx="60" cy="40" r="22" stroke={accent} strokeWidth="1" fill="none" opacity="0.8" />
      <circle cx="60" cy="40" r="16" stroke={accent} strokeWidth="0.5" fill="none" opacity="0.4" />

      {/* Pupila */}
      <circle cx="60" cy="40" r="11" fill="url(#pupilGrad)" />

      {/* Reflexo */}
      <circle cx="65" cy="34" r="3.5" fill="white" opacity="0.18" />
      <circle cx="67" cy="36" r="1.2" fill="white" opacity="0.25" />

      {/* Ponto central — o olho que tudo vê */}
      <circle cx="60" cy="40" r="3" fill={accent} filter="url(#glow)" />

      {/* Animação de pulso */}
      <circle cx="60" cy="40" r="22" stroke={accent} strokeWidth="1" fill="none" opacity="0">
        <animate attributeName="r" values="22;30;22" dur="3s" repeatCount="indefinite" />
        <animate attributeName="opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite" />
      </circle>
    </svg>
  );
}
