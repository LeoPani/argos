import { NextRequest, NextResponse } from "next/server";

const ARGOS_API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const SYSTEM_PROMPT = `Você é o Argos, um assistente especializado em Propriedade Intelectual (PI) brasileiro.

Suas especialidades:
- Patentes de Invenção (PI) e Modelos de Utilidade (MU) no INPI
- Marcas e proteção de sinais distintivos
- Prior art (anterioridade) e busca de anterioridades
- Prazos, custos e anuidades no INPI
- Transferência de tecnologia e licenciamento
- Arbitragem de disputas de PI
- Oportunidades de PI em pesquisa acadêmica (especialmente UFOP)
- Legislação brasileira de PI (Lei 9.279/1996)

Regras:
- Responda sempre em português do Brasil
- Seja conciso e objetivo — prefira listas e tópicos
- Quando relevante, sugira ações concretas (ex: "inicie uma consulta de anterioridade")
- Para estimativas de custo, use a tabela de anuidades INPI vigente
- Nunca invente números de processos ou datas reais
- Se não souber algo com certeza, diga que a informação deve ser verificada junto ao INPI ou a um agente de PI

Tabela de anuidades INPI 2024 (MPE — Micro/Pequena Empresa):
- Anos 3-6: R$ 310/ano
- Anos 7-10: R$ 620/ano
- Anos 11-15: R$ 930/ano
- Anos 16-20: R$ 1.240/ano

Tabela anuidades (demais empresas):
- Anos 3-6: R$ 785/ano
- Anos 7-10: R$ 1.570/ano
- Anos 11-15: R$ 2.355/ano
- Anos 16-20: R$ 3.140/ano`;

interface ChatRequestMsg { role: string; content: string }
interface ChatRequest {
  messages: ChatRequestMsg[];
  thread_id?: number | null;
}

// ─── Anthropic provider ──────────────────────────────────────────────────────
async function callAnthropic(messages: ChatRequestMsg[]): Promise<string> {
  const Anthropic = (await import("@anthropic-ai/sdk")).default;
  const client = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY! });
  const response = await client.messages.create({
    model: "claude-sonnet-4-6",
    max_tokens: 1024,
    system: SYSTEM_PROMPT,
    messages: messages.slice(-20).map(m => ({
      role: m.role as "user" | "assistant",
      content: m.content,
    })),
  });
  return response.content[0].type === "text" ? response.content[0].text : "";
}

// ─── Groq provider (fallback) ────────────────────────────────────────────────
async function callGroq(messages: ChatRequestMsg[]): Promise<string> {
  const resp = await fetch("https://api.groq.com/openai/v1/chat/completions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${process.env.GROQ_API_KEY}`,
    },
    body: JSON.stringify({
      model: "llama-3.3-70b-versatile",
      max_tokens: 1024,
      temperature: 0.3,
      messages: [
        { role: "system", content: SYSTEM_PROMPT },
        ...messages.slice(-20).map(m => ({ role: m.role, content: m.content })),
      ],
    }),
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({}));
    throw new Error(`Groq error ${resp.status}: ${JSON.stringify(err)}`);
  }
  const data = await resp.json();
  return data.choices?.[0]?.message?.content ?? "";
}

export async function POST(req: NextRequest) {
  try {
    const { messages, thread_id }: ChatRequest = await req.json();

    const hasAnthropic = !!process.env.ANTHROPIC_API_KEY;
    const hasGroq      = !!process.env.GROQ_API_KEY;

    if (!hasAnthropic && !hasGroq) {
      return NextResponse.json(
        { error: "Nenhuma API key configurada (ANTHROPIC_API_KEY ou GROQ_API_KEY)." },
        { status: 500 }
      );
    }

    const lastUser = [...messages].reverse().find(m => m.role === "user");
    if (!lastUser) {
      return NextResponse.json({ error: "Nenhuma mensagem do usuário" }, { status: 400 });
    }

    // Usa Anthropic se disponível, senão Groq
    let assistantText = "";
    let provider = "groq";
    if (hasAnthropic) {
      assistantText = await callAnthropic(messages);
      provider = "anthropic";
    } else {
      assistantText = await callGroq(messages);
    }

    // ─ Persistir no backend (best-effort) ─
    let threadID = thread_id ?? null;
    try {
      if (!threadID) {
        const createResp = await fetch(`${ARGOS_API}/api/v1/chat/threads`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ first_message: lastUser.content }),
        });
        if (createResp.ok) {
          const created = await createResp.json();
          threadID = created.id;
        }
      }
      if (threadID) {
        await Promise.all([
          fetch(`${ARGOS_API}/api/v1/chat/threads/${threadID}/messages`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ role: "user", content: lastUser.content }),
          }),
          fetch(`${ARGOS_API}/api/v1/chat/threads/${threadID}/messages`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ role: "assistant", content: assistantText }),
          }),
        ]);
      }
    } catch (persistErr) {
      console.warn("chat persistence failed (non-fatal):", persistErr);
    }

    return NextResponse.json({ content: assistantText, thread_id: threadID, provider });
  } catch (err) {
    console.error("chat route error:", err);
    const msg = err instanceof Error ? err.message : String(err);
    return NextResponse.json({ error: `Erro ao consultar IA: ${msg}` }, { status: 500 });
  }
}
