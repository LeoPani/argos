"use client";

import { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { useChatThreads, useChatThread } from "@/lib/hooks";
import { api } from "@/lib/api";
import type { ChatMessage as PersistedMsg, ChatRole } from "@/lib/types";
import {
  Send, MessageSquare, Sparkles, AlertCircle,
  Plus, Trash2, Clock,
} from "lucide-react";

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
}

const SUGGESTIONS = [
  "Minha ideia pode ser patenteada?",
  "Qual a diferença entre Patente de Invenção e Modelo de Utilidade?",
  "Como calcular as anuidades de uma patente no Brasil?",
  "O que é prior art e como afeta minha patente?",
  "Como funciona a transferência de tecnologia com a UFOP?",
  "Qual o prazo para responder uma exigência do INPI?",
];

const WELCOME: Message = {
  id: "welcome",
  role: "assistant",
  content: `Olá! Sou o **Argos** 👁️, seu assistente de Propriedade Intelectual.

Posso te ajudar com:
- **Patenteabilidade** de invenções e modelos de utilidade
- **Prazos e anuidades** no INPI
- **Prior art** e consulta de anterioridades
- **Marcas** e proteção de sinais distintivos
- **Transferência de tecnologia** e licenciamentos (UFOP)
- **Arbitragem** e disputas de PI

Como posso te ajudar hoje?`,
};

export default function ChatPage() {
  // Persistence state
  const { data: threadsData, mutate: refreshThreads } = useChatThreads();
  const threads = threadsData?.items ?? [];

  const [activeThreadID, setActiveThreadID] = useState<number | null>(null);
  const { data: activeThread } = useChatThread(activeThreadID);

  // Chat UI state
  const [messages, setMessages] = useState<Message[]>([WELCOME]);
  const [input, setInput]       = useState("");
  const [loading, setLoading]   = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);

  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef  = useRef<HTMLInputElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  // When a thread is selected, hydrate messages from backend.
  useEffect(() => {
    if (!activeThread?.messages) return;
    setMessages(
      activeThread.messages
        .filter(m => m.role !== "system")
        .map(m => ({
          id: String(m.id),
          role: m.role as "user" | "assistant",
          content: m.content,
        }))
    );
  }, [activeThread]);

  function newConversation() {
    setActiveThreadID(null);
    setMessages([WELCOME]);
    setApiError(null);
    setTimeout(() => inputRef.current?.focus(), 50);
  }

  async function deleteThread(id: number) {
    if (!confirm("Excluir esta conversa?")) return;
    try {
      await api.chat.deleteThread(id);
      if (id === activeThreadID) newConversation();
      refreshThreads();
    } catch (e) { console.error(e); }
  }

  async function sendMessage(text?: string) {
    const content = (text ?? input).trim();
    if (!content || loading) return;

    const userMsg: Message = { id: Date.now().toString(), role: "user", content };
    setMessages(prev => [...prev, userMsg]);
    setInput("");
    setLoading(true);
    setApiError(null);

    try {
      const history = [...messages.filter(m => m.id !== "welcome"), userMsg];

      const res = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ messages: history, thread_id: activeThreadID }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`);

      const assistantMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: data.content,
      };
      setMessages(prev => [...prev, assistantMsg]);

      // Backend created a thread (first message) — adopt the id.
      if (data.thread_id && !activeThreadID) {
        setActiveThreadID(data.thread_id);
      }
      refreshThreads();
    } catch (err) {
      setApiError(err instanceof Error ? err.message : "Erro desconhecido");
    } finally {
      setLoading(false);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }

  function renderContent(text: string) {
    return text
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(/\n- /g, "<br>• ")
      .replace(/\n\n/g, "<br><br>")
      .replace(/\n/g, "<br>");
  }

  const showSuggestions = messages.length === 1 && !loading && !activeThreadID;

  return (
    <div className="flex h-screen">
      {/* ── Sidebar with thread list ───────────────────────────────── */}
      <aside className="w-72 shrink-0 flex flex-col"
        style={{ borderRight: "1px solid var(--border)", background: "var(--surface)" }}>
        <div className="p-4" style={{ borderBottom: "1px solid var(--border)" }}>
          <Button size="sm" onClick={newConversation} className="w-full">
            <Plus size={13} />
            Nova conversa
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {threads.length === 0 && (
            <p className="text-xs text-center py-4" style={{ color: "var(--text-muted)" }}>
              Nenhuma conversa ainda.
            </p>
          )}
          {threads.map(t => {
            const isActive = activeThreadID === t.id;
            return (
              <button key={t.id}
                onClick={() => setActiveThreadID(t.id)}
                className="w-full text-left group relative px-3 py-2 rounded-lg transition-colors"
                style={{
                  background: isActive ? "var(--surface-2)" : "transparent",
                  border: `1px solid ${isActive ? "var(--accent)" : "transparent"}`,
                }}>
                <p className="text-sm text-white truncate pr-6">{t.title}</p>
                <p className="text-xs mt-0.5 flex items-center gap-1"
                  style={{ color: "var(--text-muted)" }}>
                  <Clock size={9} />
                  {new Date(t.updated_at).toLocaleDateString("pt-BR", {
                    day: "2-digit", month: "short",
                  })}
                  <span>· {t.message_count} msg</span>
                </p>
                <button
                  onClick={(e) => { e.stopPropagation(); deleteThread(t.id); }}
                  className="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-white/10"
                  aria-label="Excluir"
                  style={{ color: "#f87171" }}>
                  <Trash2 size={11} />
                </button>
              </button>
            );
          })}
        </div>
      </aside>

      {/* ── Main chat ──────────────────────────────────────────────── */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <div className="px-8 py-5 shrink-0"
          style={{ borderBottom: "1px solid var(--border)", background: "var(--surface)" }}>
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <MessageSquare size={20} />
            {activeThread?.title ?? "Chat de PI"}
            <span className="ml-1 px-2 py-0.5 text-xs rounded-full bg-indigo-500/20 text-indigo-300 flex items-center gap-1">
              <Sparkles size={10} />
              Claude Sonnet 4.6
            </span>
          </h1>
          <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
            {activeThreadID
              ? `Conversa #${activeThreadID} · ${messages.filter(m => m.id !== "welcome").length} mensagens persistidas`
              : "Assistente especializado em propriedade intelectual brasileira"}
          </p>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-8 py-6 space-y-5">
          {messages.map(msg => (
            <div key={msg.id}
              className={`flex gap-3 message-enter ${msg.role === "user" ? "flex-row-reverse" : ""}`}>
              <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 text-xs font-bold ${
                  msg.role === "assistant" ? "bg-indigo-600 text-white" : "bg-slate-700 text-white"
                }`}>
                {msg.role === "assistant" ? "👁" : "EU"}
              </div>
              <div className={`max-w-2xl px-4 py-3 rounded-2xl text-sm leading-relaxed ${
                  msg.role === "user" ? "rounded-tr-sm" : "rounded-tl-sm"
                }`}
                style={{
                  background: msg.role === "user" ? "var(--accent)" : "var(--surface)",
                  border:     msg.role === "assistant" ? "1px solid var(--border)" : "none",
                  color:      "var(--text)",
                }}
                dangerouslySetInnerHTML={{ __html: renderContent(msg.content) }} />
            </div>
          ))}

          {loading && (
            <div className="flex gap-3">
              <div className="w-8 h-8 rounded-full bg-indigo-600 flex items-center justify-center shrink-0 text-xs">👁</div>
              <div className="px-4 py-3 rounded-2xl rounded-tl-sm"
                style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
                <div className="flex gap-1 items-center">
                  {[0, 1, 2].map(i => (
                    <div key={i} className="w-1.5 h-1.5 rounded-full bg-indigo-400 animate-bounce"
                      style={{ animationDelay: `${i * 0.15}s` }} />
                  ))}
                </div>
              </div>
            </div>
          )}

          {apiError && (
            <div className="flex gap-3">
              <div className="w-8 h-8 rounded-full bg-red-600/20 flex items-center justify-center shrink-0 text-xs">
                <AlertCircle size={14} className="text-red-400" />
              </div>
              <div className="px-4 py-3 rounded-2xl rounded-tl-sm text-sm"
                style={{ background: "#7f1d1d20", border: "1px solid #ef444460", color: "#fca5a5" }}>
                {apiError}
              </div>
            </div>
          )}

          <div ref={bottomRef} />
        </div>

        {/* Suggestions (first message only) */}
        {showSuggestions && (
          <div className="px-8 pb-3 flex flex-wrap gap-2">
            {SUGGESTIONS.map(s => (
              <button key={s} onClick={() => sendMessage(s)}
                className="px-3 py-1.5 text-xs rounded-full transition-all hover:scale-105"
                style={{
                  background: "var(--surface)", border: "1px solid var(--border)",
                  color: "var(--text-muted)",
                }}>
                {s}
              </button>
            ))}
          </div>
        )}

        {/* Input */}
        <div className="px-8 py-4 shrink-0"
          style={{ borderTop: "1px solid var(--border)", background: "var(--surface)" }}>
          <form onSubmit={(e) => { e.preventDefault(); sendMessage(); }} className="flex gap-2">
            <input ref={inputRef} value={input} onChange={(e) => setInput(e.target.value)}
              placeholder={activeThreadID ? "Continue a conversa…" : "Pergunte sobre patentes, marcas, prazos, custos…"}
              disabled={loading}
              className="flex-1 px-4 py-2.5 rounded-lg text-sm outline-none transition-colors"
              style={{
                background: "var(--surface-2)", border: "1px solid var(--border)", color: "white",
              }}
            />
            <Button type="submit" disabled={loading || !input.trim()}>
              <Send size={14} />
              {loading ? "Pensando..." : "Enviar"}
            </Button>
          </form>
        </div>
      </div>

      <style jsx global>{`
        .message-enter {
          animation: msgIn 0.3s ease-out;
        }
        @keyframes msgIn {
          from { opacity: 0; transform: translateY(8px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}
