import { NextRequest, NextResponse } from "next/server";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Proxy GET /api/prior-art → Go backend /api/v1/prior-art
// Needed to avoid CORS issues when the backend is on a different port in prod.
export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const q = searchParams.get("q") ?? "";
  const kind = searchParams.get("kind") ?? "both";

  try {
    const res = await fetch(
      `${API}/api/v1/prior-art?q=${encodeURIComponent(q)}&kind=${kind}`,
      { headers: { "Content-Type": "application/json" } }
    );
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch {
    return NextResponse.json({ error: "Backend offline" }, { status: 503 });
  }
}
