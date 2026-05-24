"use client";
// SWR hooks for all API endpoints.
// Each hook returns { data, error, isLoading } from SWR with
// automatic revalidation on focus and a 30-second interval.

import useSWR from "swr";
import { api } from "./api";
import type { Patent, PatentListResponse, UFOPListResponse } from "./types";

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

// ─── Health ───────────────────────────────────────────────────────────────────

export function useHealth() {
  return useSWR<{ status: string }>(
    "/health",
    () => api.health.check(),
    { refreshInterval: 10_000 }
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
