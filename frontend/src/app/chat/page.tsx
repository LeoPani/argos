"use client";

import { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Send, MessageSquare, Sparkles, AlertCircle } from "lucide-react";

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
  const [messages, setMessages] = useState<Message[]>([WELCOME]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  async function sendMessage(text?: string) {
    const content = (text ?? input).trim();
    if (!content || loading) return;

    const userMsg: Message = {
      id: Date.now().toString(),
      role: "user",
      content,
    };

    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setLoading(true);
    setApiError(null);

    try {
      // Build history excluding the welcome message
      const history = [...messages.filter((m) => m.id !== "welcome"), userMsg];

      const res = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ messages: history }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.error ?? `HTTP ${res.status}`);
      }

      const assistantMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: data.content,
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro desconhecido";
      setApiError(msg);
    } finally {
      setLoading(false);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }

  // Simple markdown-ish renderer
  function renderContent(text: string) {
    return text
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(/\n- /g, "<br>• ")
      .replace(/\n\n/g, "<br><br>")
      .replace(/\n/g, "<br>");
  }

  const showSuggestions = messages.length === 1 && !loading;

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div
        className="px-8 py-5 shrink-0"
        style={{
          borderBottom: "1px solid var(--border)",
          background: "var(--surface)",
        }}
      >
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <MessageSquare size={20} />
          Chat de PI
          <span className="ml-1 px-2 py-0.5 text-xs rounded-full bg-indigo-500/20 text-indigo-300 flex items-center gap-1">
            <Sparkles size={10} />
            Claude Sonnet
          </span>
        </h1>
        <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
          Assistente especializado em propriedade intelectual brasileira
        </p>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-8 py-6 space-y-5">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex gap-3 ${msg.role === "user" ? "flex-row-reverse" : ""}`}
          >
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 text-xs font-bold ${
                msg.role === "assistant"
                  ? "bg-indigo-600 text-white"
                  : "bg-slate-700 text-white"
              }`}
            >
              {msg.role === "assistant" ? "👁" : "EU"}
            </div>
            <div
              className={`max-w-2xl px-4 py-3 rounded-2xl text-sm leading-relaxed ${
                msg.role === "user" ? "rounded-tr-sm" : "rounded-tl-sm"
              }`}
              style={{
                background:
                  msg.role === "user" ? "var(--accent)" : "var(--surface)",
                border:
                  msg.role === "assistant"
                    ? "1px solid var(--border)"
                    : "none",
                color: "var(--text)",
              }}
              dangerouslySetInnerHTML={{
                __html: renderContent(msg.content),
              }}
            />
          </div>
        ))}

        {/* Typing indicator */}
        {loading && (
          <div className="flex gap-3">
            <div className="w-8 h-8 rounded-full bg-indigo-600 flex items-center justify-center shrink-0 text-xs">
              👁
            </div>
            <div
              className="px-4 py-3 rounded-2xl rounded-tl-sm"
              style={{
                background: "var(--surface)",
                border: "1px solid var(--border)",
              }}
            >
              <div className="flex gap-1 items-center">
                {[0, 1, 2].map((i) => (
                  <div
                    key={i}
                    className="w-1.5 h-1.5 rounded-full bg-indigo-400 animate-bounce"
                    style={{ animationDelay: `${i * 0.15}s` }}
                  />
                ))}
              </div>
            </div>
          </div>
        )}

        {/* API error */}
        {apiError && (
          <div
            className="flex items-center gap-2 px-4 py-3 rounded-xl text-sm"
            style={{
              background: "#ef444420",
              border: "1px solid #ef444440",
              color: "#fca5a5",
            }}
          >
            <AlertCircle size={14} />
            {apiError.includes("ANTHROPIC_API_KEY")
              ? "Configure a ANTHROPIC_API_KEY em .env.local para usar o chat."
              : `Erro: ${apiError}`}
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Suggestions */}
      {showSuggestions && (
        <div className="px-8 pb-3 flex gap-2 flex-wrap">
          {SUGGESTIONS.map((s) => (
            <button
              key={s}
              onClick={() => sendMessage(s)}
              className="text-xs px-3 py-1.5 rounded-full transition-all"
              style={{
                background: "var(--surface)",
                border: "1px solid var(--border)",
                color: "var(--text-muted)",
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  "var(--accent)";
                (e.currentTarget as HTMLElement).style.color = "white";
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLElement).style.borderColor =
                  "var(--border)";
                (e.currentTarget as HTMLElement).style.color =
                  "var(--text-muted)";
              }}
            >
              {s}
            </button>
          ))}
        </div>
      )}

      {/* Input */}
      <div
        className="px-8 py-4 shrink-0"
        style={{
          borderTop: "1px solid var(--border)",
          background: "var(--surface)",
        }}
      >
        <div className="flex gap-3">
          <input
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
              }
            }}
            placeholder="Pergunte sobre patentes, marcas, prazos, custos..."
            className="flex-1 px-4 py-3 rounded-xl text-sm outline-none focus:ring-1 focus:ring-indigo-500"
            style={{
              background: "var(--surface-2)",
              border: "1px solid var(--border)",
              color: "white",
            }}
            disabled={loading}
          />
          <Button
            onClick={() => sendMessage()}
            disabled={!input.trim() || loading}
            size="lg"
          >
            <Send size={15} />
          </Button>
        </div>
        <p className="text-xs mt-2 text-center" style={{ color: "var(--text-muted)" }}>
          Powered by Claude Sonnet · não substitui consulta jurídica profissional
        </p>
      </div>
    </div>
  );
}
