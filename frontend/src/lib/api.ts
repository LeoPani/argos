// ─── Argos API client ─────────────────────────────────────────────────────────
// Connects to the Go backend at :8080

const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${res.status}: ${body}`);
  }
  return res.json() as Promise<T>;
}

// ─── Patents ──────────────────────────────────────────────────────────────────

import type {
  Patent, PatentListResponse,
  UFOPListResponse, UFOPOpportunity, UFOPStatus,
  PortfolioResponse,
  StatsResponse,
  WatchType, Watchlist, WatchlistListResponse,
  Dispute, DisputeListResponse, DisputeKind, DisputeStatus,
} from "./types";

export const api = {
  patents: {
    list: (params?: Record<string, string>) => {
      const qs = params ? "?" + new URLSearchParams(params).toString() : "";
      return req<PatentListResponse>(`/api/v1/patents${qs}`);
    },
    get: (id: number) => req<Patent>(`/api/v1/patents/${id}`),
    create: (body: Partial<Patent>) =>
      req<Patent>("/api/v1/patents", {
        method: "POST",
        body: JSON.stringify(body),
      }),
  },

  health: {
    check: () => req<{ status: string }>("/health"),
  },

  portfolio: {
    get: () => req<PortfolioResponse>("/api/v1/portfolio"),
  },

  stats: {
    get: () => req<StatsResponse>("/api/v1/stats"),
  },

  watchlists: {
    list:   () => req<WatchlistListResponse>("/api/v1/watchlists"),
    create: (body: { label: string; watch_type: WatchType; query?: string }) =>
      req<Watchlist>("/api/v1/watchlists", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      fetch(`${BASE}/api/v1/watchlists/${id}`, { method: "DELETE" }).then(r => {
        if (!r.ok) throw new Error(`API ${r.status}`);
      }),
    check: (id: number) =>
      req<Watchlist>(`/api/v1/watchlists/${id}/check`, { method: "POST" }),
    checkAll: () =>
      req<{ checked: number }>("/api/v1/watchlists/check-all", { method: "POST" }),
  },

  disputes: {
    list: (params?: Record<string, string>) => {
      const qs = params ? "?" + new URLSearchParams(params).toString() : "";
      return req<DisputeListResponse>(`/api/v1/disputes${qs}`);
    },
    get: (id: number) => req<Dispute>(`/api/v1/disputes/${id}`),
    open: (body: { case_number: string; title: string; summary: string; kind: DisputeKind }) =>
      req<Dispute>("/api/v1/disputes", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateStatus: (id: number, status: DisputeStatus) =>
      req<{ ok: boolean }>(`/api/v1/disputes/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      }),
  },

  ufop: {
    list: (params?: Record<string, string>) => {
      const qs = params ? "?" + new URLSearchParams(params).toString() : "";
      return req<UFOPListResponse>(`/api/v1/ufop/opportunities${qs}`);
    },
    get: (id: number) => req<UFOPOpportunity>(`/api/v1/ufop/opportunities/${id}`),
    updateStatus: (id: number, status: UFOPStatus) =>
      req<{ status: string }>(`/api/v1/ufop/opportunities/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      }),
  },
};

// ─── Mock helpers (used while backend endpoints don't exist yet) ───────────────

export function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}
