import { NextRequest, NextResponse } from "next/server";

const ACCESS_KEY = process.env.ARGOS_ACCESS_KEY ?? "";

export async function POST(req: NextRequest) {
  const { key } = await req.json().catch(() => ({ key: "" }));

  if (!ACCESS_KEY) {
    return NextResponse.json(
      { error: "ARGOS_ACCESS_KEY não configurada no servidor." },
      { status: 500 }
    );
  }

  if (!key || key.trim() !== ACCESS_KEY.trim()) {
    // Delay para dificultar brute-force
    await new Promise((r) => setTimeout(r, 600));
    return NextResponse.json({ error: "Chave de acesso inválida." }, { status: 401 });
  }

  const res = NextResponse.json({ ok: true });
  res.cookies.set("argos-session", ACCESS_KEY, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    // Em produção, adicionar: secure: true, maxAge: 60 * 60 * 24 * 30
    maxAge: 60 * 60 * 24 * 30, // 30 dias
  });
  return res;
}

export async function DELETE() {
  const res = NextResponse.json({ ok: true });
  res.cookies.delete("argos-session");
  return res;
}
