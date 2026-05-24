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
  Trademark,
  UFOPListResponse, UFOPOpportunity, UFOPStatus,
  PortfolioResponse,
  StatsResponse,
  WatchType, Watchlist, WatchlistListResponse,
  Dispute, DisputeListResponse, DisputeKind, DisputeStatus,
  DisputeSubject, SubjectKind, ArbitrationVerdict,
  TTContract, TTContractListResponse, LicenseKind, ContractStatus,
  PatentPool, PoolListResponse, PoolKind, PoolMember,
  ChatThread, ChatThreadListResponse, ChatRole, ChatMessage,
  SearchResponse,
  MetricsResponse, MethodologyPayload, PCIScore,
  MaintenanceRecommendation, InventorProfile,
  DepartmentHealth, KnowledgeStockResponse,
  RoyaltyForecast, FilingSuggestion,
  MarketplaceResponse, CitationNetwork,
  CalendarResponse,
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

  trademarks: {
    get: (id: number) => req<Trademark>(`/api/v1/trademarks/${id}`),
  },

  health: {
    check: () => req<{ status: string }>("/health"),
  },

  search: {
    query: (q: string, limit = 6) =>
      req<SearchResponse>(`/api/v1/search?q=${encodeURIComponent(q)}&limit=${limit}`),
  },

  metrics: {
    snapshot: (scope = "UFOP") =>
      req<MetricsResponse>(`/api/v1/metrics?scope=${encodeURIComponent(scope)}`),
    methodology: () => req<MethodologyPayload>("/api/v1/metrics/methodology"),
    pci: (patentID: number) => req<PCIScore>(`/api/v1/metrics/patent/${patentID}/pci`),
    maintenance: (patentID: number) =>
      req<MaintenanceRecommendation>(`/api/v1/metrics/patent/${patentID}/maintenance`),
    inventor: (name: string) =>
      req<InventorProfile>(`/api/v1/metrics/inventors/${encodeURIComponent(name)}`),
    departments: () =>
      req<{ departments: DepartmentHealth[] }>("/api/v1/metrics/departments"),
    knowledgeStock: (scope = "UFOP") =>
      req<KnowledgeStockResponse>(`/api/v1/metrics/knowledge-stock?scope=${encodeURIComponent(scope)}`),
    royaltyForecast: (years = 10) =>
      req<RoyaltyForecast>(`/api/v1/metrics/royalty-forecast?years=${years}`),
    enrichAll: (limit = 50) =>
      req<{ processed: number; errors: number; source: string; avg_fwd_citations: number }>(
        `/api/v1/metrics/enrich-all?limit=${limit}`,
        { method: "POST" },
      ),
    enrichOne: (patentID: number) =>
      req<{ patent_id: number; lens_id: string; forward_citations: number; backward_citations: number; family_size: number; claims_count: number; source: string }>(
        `/api/v1/metrics/enrich/${patentID}`,
        { method: "POST" },
      ),
  },

  smartFiling: {
    analyze: (body: { title: string; abstract: string; field?: string }) =>
      req<FilingSuggestion>("/api/v1/smart-filing", {
        method: "POST",
        body: JSON.stringify(body),
      }),
  },

  marketplace: {
    list: (params?: { ipc?: string; q?: string; limit?: number }) => {
      const p = new URLSearchParams();
      if (params?.ipc)   p.set("ipc",   params.ipc);
      if (params?.q)     p.set("q",     params.q);
      if (params?.limit) p.set("limit", String(params.limit));
      const qs = p.toString();
      return req<MarketplaceResponse>(`/api/v1/marketplace${qs ? "?" + qs : ""}`);
    },
  },

  citations: {
    network: (patentID: number) =>
      req<CitationNetwork>(`/api/v1/citations/network/${patentID}`),
  },

  calendar: {
    get: (from?: string, to?: string) => {
      const p = new URLSearchParams();
      if (from) p.set("from", from);
      if (to)   p.set("to",   to);
      const qs = p.toString();
      return req<CalendarResponse>(`/api/v1/calendar${qs ? "?" + qs : ""}`);
    },
  },

  portfolio: {
    get: () => req<PortfolioResponse>("/api/v1/portfolio"),
  },

  stats: {
    get: () => req<StatsResponse>("/api/v1/stats"),
  },

  watchlists: {
    list:   () => req<WatchlistListResponse>("/api/v1/watchlists"),
    create: (body: { label: string; watch_type: WatchType; query?: string; auto_dispute?: boolean; similarity_threshold?: number }) =>
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

    // ─ Subjects + AI verdict ─
    listSubjects: (disputeID: number) =>
      req<{ items: DisputeSubject[]; count: number }>(`/api/v1/disputes/${disputeID}/subjects`),
    addSubject: (disputeID: number, body: { kind: SubjectKind; ref_id?: number; label: string }) =>
      req<DisputeSubject>(`/api/v1/disputes/${disputeID}/subjects`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    removeSubject: (subjectID: number) =>
      fetch(`${BASE}/api/v1/disputes/subjects/${subjectID}`, { method: "DELETE" })
        .then(r => { if (!r.ok) throw new Error(`API ${r.status}`); }),
    analyze: (disputeID: number) =>
      req<ArbitrationVerdict>(`/api/v1/disputes/${disputeID}/analyze`, { method: "POST" }),
    verdict: (disputeID: number) =>
      req<{ verdict: ArbitrationVerdict | null }>(`/api/v1/disputes/${disputeID}/verdict`),
  },

  ttContracts: {
    list: (params?: Record<string, string>) => {
      const qs = params ? "?" + new URLSearchParams(params).toString() : "";
      return req<TTContractListResponse>(`/api/v1/tt-contracts${qs}`);
    },
    get: (id: number) => req<TTContract>(`/api/v1/tt-contracts/${id}`),
    create: (body: Partial<TTContract>) =>
      req<TTContract>("/api/v1/tt-contracts", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateStatus: (id: number, status: ContractStatus) =>
      req<{ ok: boolean }>(`/api/v1/tt-contracts/${id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      }),
    delete: (id: number) =>
      fetch(`${BASE}/api/v1/tt-contracts/${id}`, { method: "DELETE" })
        .then(r => { if (!r.ok) throw new Error(`API ${r.status}`); }),
  },

  chat: {
    listThreads: () => req<ChatThreadListResponse>("/api/v1/chat/threads"),
    createThread: (firstMessage: string) =>
      req<ChatThread>("/api/v1/chat/threads", {
        method: "POST",
        body: JSON.stringify({ first_message: firstMessage }),
      }),
    getThread: (id: number) => req<ChatThread>(`/api/v1/chat/threads/${id}`),
    deleteThread: (id: number) =>
      fetch(`${BASE}/api/v1/chat/threads/${id}`, { method: "DELETE" })
        .then(r => { if (!r.ok) throw new Error(`API ${r.status}`); }),
    appendMessage: (threadID: number, role: ChatRole, content: string) =>
      req<ChatMessage>(`/api/v1/chat/threads/${threadID}/messages`, {
        method: "POST",
        body: JSON.stringify({ role, content }),
      }),
    updateTitle: (id: number, title: string) =>
      req<{ title: string }>(`/api/v1/chat/threads/${id}/title`, {
        method: "PATCH",
        body: JSON.stringify({ title }),
      }),
  },

  pools: {
    list: () => req<PoolListResponse>("/api/v1/pools"),
    get:  (id: number) => req<PatentPool>(`/api/v1/pools/${id}`),
    create: (body: { name: string; description?: string; pool_kind?: PoolKind; royalty_rate?: number; duration_years?: number }) =>
      req<PatentPool>("/api/v1/pools", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      fetch(`${BASE}/api/v1/pools/${id}`, { method: "DELETE" })
        .then(r => { if (!r.ok) throw new Error(`API ${r.status}`); }),
    addMember: (poolID: number, body: { patent_id: number; share_pct: number }) =>
      req<PoolMember>(`/api/v1/pools/${poolID}/members`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    removeMember: (poolID: number, patentID: number) =>
      fetch(`${BASE}/api/v1/pools/${poolID}/members/${patentID}`, { method: "DELETE" })
        .then(r => { if (!r.ok) throw new Error(`API ${r.status}`); }),
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
