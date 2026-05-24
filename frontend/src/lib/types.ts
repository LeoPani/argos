// ─── Domain types (mirror Go backend) ───────────────────────────────────────

export type PatentStatus = "pending" | "classified" | "failed";

export interface Patent {
  id: number;
  application_number: string;
  title: string;
  abstract: string;
  applicant: string;
  inventors: string[];
  filing_date: string | null;
  publication_date: string | null;
  ipc_category: number | null;
  ipc_code: string;
  rpi_issue: string;
  status: PatentStatus;
  created_at: string;
  updated_at: string;
}

export interface PatentListResponse {
  items: Patent[];
  pagination: { total: number; limit: number; offset: number };
}

export type TrademarkStatus = "active" | "pending" | "opposition" | "expired";

export interface Trademark {
  id: number;
  number: string;
  name: string;
  owner: string;
  nice_class: number;
  status: TrademarkStatus;
  filing_date: string | null;
  expiry_date: string | null;
  cost_annual: number;
}

// ─── Portfolio ───────────────────────────────────────────────────────────────

export type AssetType = "PI" | "MU" | "TM" | "DP";
export type AssetStatus = "active" | "pending" | "expired" | "opposition";

export interface PortfolioAsset {
  id: string;
  type: AssetType;
  number: string;
  title: string;
  owner: string;
  status: AssetStatus;
  filing_date: string;
  expiry_date: string;
  next_fee_date: string | null;
  cost_annual: number;
  cost_monthly: number;
  cost_total: number;
  ipc_code?: string;
  blockchain_hash?: string;
}

export interface CostSummary {
  monthly: number;
  annual: number;
  total: number;
}

export interface CostPoint {
  year: string;
  value: number;
}

export interface AIOpportunity {
  id: string;
  type: "opportunity" | "risk";
  title: string;
  description: string;
  ipc_class?: string;
  estimated_cost?: number;
  confidence: number; // 0-100
  action_label: string;
}

/** Full response from GET /api/v1/portfolio */
export interface PortfolioResponse {
  assets: PortfolioAsset[];
  cost_summary: CostSummary;
  cost_timeline: CostPoint[];
  ai_opportunities: AIOpportunity[];
}

// ─── Arbitration ─────────────────────────────────────────────────────────────

export type DisputeStatus = "open" | "in_analysis" | "mediation" | "resolved" | "urgent";

export interface Dispute {
  id: string;
  number: string;
  title: string;
  plaintiff: string;
  defendant: string;
  status: DisputeStatus;
  opened_at: string;
  deadline_days: number;
  blockchain_hash?: string;
}

// ─── UFOP Intelligence ────────────────────────────────────────────────────────

export type OpportunityLevel = "high" | "medium" | "low";
export type UFOPSource = "oai" | "portal" | "lens";
export type UFOPStatus = "new" | "reviewed" | "converted" | "dismissed";

/** Mirrors domain.UFOPOpportunity from the Go backend. */
export interface UFOPOpportunity {
  id: number;
  source: UFOPSource;
  external_id: string;
  title: string;
  authors: string[];
  department: string;
  abstract: string;
  url: string;
  published_at: string | null;

  // AI analysis
  ipc_suggestion: string;
  ipc_category: number;
  opportunity_level: OpportunityLevel;
  similarity_pct: number;
  pi_score: number;
  ai_analysis: string;

  // Lifecycle
  status: UFOPStatus;
  publication_id: number | null;

  created_at: string;
  updated_at: string;
}

export interface UFOPListResponse {
  items: UFOPOpportunity[];
  pagination: { total: number; limit: number; offset: number };
}

export interface UFOPNews {
  id: string;
  title: string;
  date: string;
  url: string;
  pi_keywords: string[];
}

// ─── Patent Pool & TT ────────────────────────────────────────────────────────

export type LicenseType = "exclusive" | "non-exclusive" | "sub-licensable";
export type ContractStatus = "active" | "negotiating" | "expired" | "draft";

export interface PoolPatent {
  id: string;
  number: string;
  title: string;
  department: string;
  ipc_code: string;
  license_type: LicenseType;
  royalty_suggestion: string;
  ai_match: string;
  prospectus_url?: string;
}

export interface TTContract {
  id: string;
  number: string;
  patent_title: string;
  licensor: string;
  licensee: string;
  status: ContractStatus;
  signed_at: string;
  expiry_at: string;
  royalty_rate: number;
  milestones: Milestone[];
  royalties: RoyaltyEntry[];
  blockchain_hash?: string;
}

export interface Milestone {
  label: string;
  due_date: string;
  done: boolean;
}

export interface RoyaltyEntry {
  period: string;
  expected: number;
  received: number | null;
  status: "received" | "pending" | "upcoming";
}

// ─── Alerts / Watchlists ─────────────────────────────────────────────────────

export type WatchType = "term" | "brand" | "company" | "patent";
export type WatchStatus = "ok" | "alert";

/** Legacy frontend-only Alert type (still used by mock data). */
export interface Alert {
  id: string;
  type: WatchType;
  label: string;
  last_check: string;
  new_count: number;
  status: WatchStatus;
}

/** Mirrors domain.Watchlist from the Go backend. */
export interface Watchlist {
  id: number;
  label: string;
  watch_type: WatchType;
  query: string;
  last_check: string | null;
  new_count: number;
  status: WatchStatus;
  created_at: string;
  updated_at: string;
}

export interface WatchlistListResponse {
  items: Watchlist[];
  count: number;
}

// ─── Dashboard / Stats ───────────────────────────────────────────────────────

export interface StatsCounts {
  patents: number;
  patents_classified: number;
  trademarks: number;
  trademarks_active: number;
  disputes: number;
  disputes_open: number;
  ufop_opportunities: number;
  ufop_high: number;
}

export interface IPCSlice {
  category: number;
  letter: string;
  name: string;
  count: number;
  pct: number;
}

export interface StatusSlice {
  status: string;
  count: number;
  pct: number;
}

export interface ActivityItem {
  kind: "patent" | "trademark" | "dispute" | "ufop";
  id: number;
  reference: string;
  title: string;
  status: string;
  created_at: string;
}

export interface StatsResponse {
  counts: StatsCounts;
  ipc_distribution: IPCSlice[];
  patent_statuses: StatusSlice[];
  trademark_statuses: StatusSlice[];
  recent_activity: ActivityItem[];
  generated_at: string;
}

// ─── Risk / Anterioridade ────────────────────────────────────────────────────

export interface SearchResult {
  query: string;
  type: "patent" | "trademark" | "both";
  risk_score: number; // 0-10
  risk_label: "Baixo" | "Médio" | "Alto" | "Muito Alto";
  conflicts: Conflict[];
}

export interface Conflict {
  number: string;
  title: string;
  similarity_pct: number;
  owner: string;
  filing_date: string;
}
