import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PATHS = ["/login", "/api/auth/login"];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Deixar passar rotas públicas e assets
  if (
    PUBLIC_PATHS.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith("/_next") ||
    pathname.startsWith("/favicon")
  ) {
    return NextResponse.next();
  }

  const session = request.cookies.get("argos-session")?.value;
  const accessKey = process.env.ARGOS_ACCESS_KEY ?? "";

  // Sem sessão ou sessão inválida → redireciona para login
  if (!session || session !== accessKey) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("from", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Aplicar a todas as rotas exceto:
     * - _next/static, _next/image, favicon.ico
     */
    "/((?!_next/static|_next/image|favicon.ico).*)",
  ],
};
