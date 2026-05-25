"use client";
// SWR hooks for all API endpoints.
// Each hook returns { data, error, isLoading } from SWR with
// automatic revalidation on focus and a 30-second interval.

import useSWR from "swr";
import { api } from "./api";
import type {
  Patent, PatentListResponse, Trademark,
  UFOPListResponse, PortfolioResponse,
  StatsResponse, WatchlistListResponse,
  DisputeListResponse,
  DisputeSubject, ArbitrationVerdict,
  TTContractListResponse,
  PoolListResponse, PatentPool,
  ChatThreadListResponse, ChatThread,
  MetricsResponse, MethodologyPayload,
  MaintenanceRecommendation, InventorProfile,
  DepartmentHealth, KnowledgeStockResponse,
  RoyaltyForecast,
  MarketplaceResponse, CitationNetwork,
  CalendarResponse, TTTemplate,
} from "./types";

const SWR_OPTIONS = {
  refreshInterval: 30_000,
  revalidateOnFocus: true,
};

// ─── Patents ──────────────────────────────────────────────────────────────────

export function usePatents(params?: Record<string, string>) {
  const key = params
    ? ["/api/v1/patents", JSON.stringify(params)]
    : "/api/v1/patents";

  return useSWR<PatentListResponse>(
    key,
    () => api.patents.list(params),
    SWR_OPTIONS
  );
}

export function usePatent(id: number | null) {
  return useSWR<Patent>(
    id ? `/api/v1/patents/${id}` : null,
    () => api.patents.get(id!),
    SWR_OPTIONS
  );
}

export function useTrademark(id: number | null) {
  return useSWR<Trademark>(
    id ? `/api/v1/trademarks/${id}` : null,
    () => api.trademarks.get(id!),
    SWR_OPTIONS
  );
}

// ─── Health ───────────────────────────────────────────────────────────────────

export function useHealth() {
  return useSWR<{ status: string }>(
    "/health",
    () => api.health.check(),
    { refreshInterval: 10_000 }
  );
}

// ─── Portfolio ────────────────────────────────────────────────────────────────

export function usePortfolio() {
  return useSWR<PortfolioResponse>(
    "/api/v1/portfolio",
    () => api.portfolio.get(),
    SWR_OPTIONS
  );
}

// ─── Stats / Dashboard ───────────────────────────────────────────────────────

export function useStats() {
  return useSWR<StatsResponse>(
    "/api/v1/stats",
    () => api.stats.get(),
    SWR_OPTIONS
  );
}

// ─── Disputes / Arbitration ──────────────────────────────────────────────────

export function useDisputes(params?: Record<string, string>) {
  const key = params
    ? ["/api/v1/disputes", JSON.stringify(params)]
    : "/api/v1/disputes";

  return useSWR<DisputeListResponse>(
    key,
    () => api.disputes.list(params),
    SWR_OPTIONS
  );
}

// ─── Dispute subjects + verdict ──────────────────────────────────────────────

export function useDisputeSubjects(disputeID: number | null) {
  return useSWR<{ items: DisputeSubject[]; count: number }>(
    disputeID ? `/api/v1/disputes/${disputeID}/subjects` : null,
    () => api.disputes.listSubjects(disputeID!),
    SWR_OPTIONS
  );
}

export function useDisputeVerdict(disputeID: number | null) {
  return useSWR<{ verdict: ArbitrationVerdict | null }>(
    disputeID ? `/api/v1/disputes/${disputeID}/verdict` : null,
    () => api.disputes.verdict(disputeID!),
    SWR_OPTIONS
  );
}

// ─── TT Contracts + Pools ────────────────────────────────────────────────────

export function useTTContracts(params?: Record<string, string>) {
  const key = params
    ? ["/api/v1/tt-contracts", JSON.stringify(params)]
    : "/api/v1/tt-contracts";

  return useSWR<TTContractListResponse>(
    key,
    () => api.ttContracts.list(params),
    SWR_OPTIONS
  );
}

export function usePools() {
  return useSWR<PoolListResponse>(
    "/api/v1/pools",
    () => api.pools.list(),
    SWR_OPTIONS
  );
}

export function usePool(id: number | null) {
  return useSWR<PatentPool>(
    id ? `/api/v1/pools/${id}` : null,
    () => api.pools.get(id!),
    SWR_OPTIONS
  );
}

// ─── Chat threads ────────────────────────────────────────────────────────────

export function useChatThreads() {
  return useSWR<ChatThreadListResponse>(
    "/api/v1/chat/threads",
    () => api.chat.listThreads(),
    SWR_OPTIONS
  );
}

export function useChatThread(id: number | null) {
  return useSWR<ChatThread>(
    id ? `/api/v1/chat/threads/${id}` : null,
    () => api.chat.getThread(id!),
    { revalidateOnFocus: false }
  );
}

// ─── Academic metrics ────────────────────────────────────────────────────────

export function useMetrics(scope = "UFOP") {
  return useSWR<MetricsResponse>(
    `/api/v1/metrics?scope=${scope}`,
    () => api.metrics.snapshot(scope),
    SWR_OPTIONS
  );
}

export function useMethodology() {
  return useSWR<MethodologyPayload>(
    "/api/v1/metrics/methodology",
    () => api.metrics.methodology(),
    { revalidateOnFocus: false }
  );
}

export function useMaintenance(patentID: number | null) {
  return useSWR<MaintenanceRecommendation>(
    patentID ? `/api/v1/metrics/patent/${patentID}/maintenance` : null,
    () => api.metrics.maintenance(patentID!),
    SWR_OPTIONS
  );
}

export function useInventorProfile(name: string | null) {
  return useSWR<InventorProfile>(
    name ? `/api/v1/metrics/inventors/${name}` : null,
    () => api.metrics.inventor(name!),
    SWR_OPTIONS
  );
}

export function useDepartments() {
  return useSWR<{ departments: DepartmentHealth[] }>(
    "/api/v1/metrics/departments",
    () => api.metrics.departments(),
    SWR_OPTIONS
  );
}

export function useKnowledgeStock(scope = "UFOP") {
  return useSWR<KnowledgeStockResponse>(
    `/api/v1/metrics/knowledge-stock?scope=${scope}`,
    () => api.metrics.knowledgeStock(scope),
    SWR_OPTIONS
  );
}

export function useRoyaltyForecast(years = 10) {
  return useSWR<RoyaltyForecast>(
    `/api/v1/metrics/royalty-forecast?years=${years}`,
    () => api.metrics.royaltyForecast(years),
    SWR_OPTIONS
  );
}

// ─── Marketplace + Citation Network ──────────────────────────────────────────

export function useMarketplace(params?: { ipc?: string; q?: string; limit?: number }) {
  const key = `/api/v1/marketplace:${JSON.stringify(params ?? {})}`;
  return useSWR<MarketplaceResponse>(key, () => api.marketplace.list(params), SWR_OPTIONS);
}

export function useCitationNetwork(patentID: number | null) {
  return useSWR<CitationNetwork>(
    patentID ? `/api/v1/citations/network/${patentID}` : null,
    () => api.citations.network(patentID!),
    SWR_OPTIONS
  );
}

export function useCalendar(from?: string, to?: string) {
  const key = `/api/v1/calendar:${from ?? ""}:${to ?? ""}`;
  return useSWR<CalendarResponse>(key, () => api.calendar.get(from, to), SWR_OPTIONS);
}

export function useTTTemplateFromUFOP(oppID: number | null) {
  return useSWR<TTTemplate>(
    oppID ? `/api/v1/tt-template/from-ufop/${oppID}` : null,
    () => api.ttTemplate.fromUFOP(oppID!),
    { revalidateOnFocus: false }
  );
}

// ─── Watchlists / Alerts ─────────────────────────────────────────────────────

export function useWatchlists() {
  return useSWR<WatchlistListResponse>(
    "/api/v1/watchlists",
    () => api.watchlists.list(),
    SWR_OPTIONS
  );
}

// ─── UFOP Intelligence ────────────────────────────────────────────────────────

export function useUFOPOpportunities(params?: Record<string, string>) {
  const key = params
    ? ["/api/v1/ufop/opportunities", JSON.stringify(params)]
    : "/api/v1/ufop/opportunities";

  return useSWR<UFOPListResponse>(
    key,
    () => api.ufop.list(params),
    SWR_OPTIONS
  );
}
