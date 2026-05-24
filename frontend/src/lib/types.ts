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

export type TrademarkStatus =
  | "filed" | "published" | "granted"
  | "denied" | "archived" | "expired";

export type TrademarkKind =
  | "nominative" | "figurative" | "mixed" | "three_dimensional";

/** Mirrors domain.Trademark from the Go backend. */
export interface Trademark {
  id: number;
  process_number: string;
  name: string;
  normalized_name: string;
  kind: TrademarkKind;
  status: TrademarkStatus;
  owner: string;
  nice_classes: number[];
  image_url: string;
  filing_date: string | null;
  publication_date: string | null;
  granted_date: string | null;
  rpi_issue: string;
  created_at: string;
  updated_at: string;
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

/** Backend statuses — mirror domain.DisputeStatus from Go. */
export type DisputeStatus =
  | "open" | "in_review" | "awaiting_info"
  | "resolved" | "withdrawn" | "escalated";

export type DisputeKind =
  | "trademark_infringement"
  | "patent_infringement"
  | "authorship"
  | "licensing"
  | "other";

/** Mirrors domain.Dispute from the Go backend. */
export interface Dispute {
  id: number;
  case_number: string;
  title: string;
  summary: string;
  kind: DisputeKind;
  status: DisputeStatus;
  patent_id?: number | null;
  trademark_id?: number | null;
  opened_at: string;
  resolved_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface DisputeListResponse {
  items: Dispute[];
  pagination: { total: number; limit: number; offset: number };
}

// ─── Arbitration subjects + verdicts ─────────────────────────────────────────

export type SubjectKind = "trademark" | "patent" | "inventor" | "other";

export interface DisputeSubject {
  id: number;
  dispute_id: number;
  kind: SubjectKind;
  ref_id?: number | null;
  label: string;
  party_id?: number | null;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface SubjectScore {
  subject_id: number;
  label: string;
  score: number;
  pro: string[];
  con: string[];
}

export interface VerdictReasoning {
  subjects: SubjectScore[];
  factors: string[];
}

export type VerdictMethod = "heuristic_v1" | "claude_v1" | "hybrid";

export interface ArbitrationVerdict {
  id: number;
  dispute_id: number;
  winner_subject_id: number | null;
  confidence: number;
  method: VerdictMethod;
  summary: string;
  reasoning: VerdictReasoning;
  created_at: string;
}

// ─── TT Contracts ────────────────────────────────────────────────────────────

export type LicenseKind = "exclusive" | "non_exclusive" | "sole";
export type ContractStatus = "draft" | "negotiating" | "active" | "expired" | "terminated";

export interface Milestone {
  label: string;
  due_date?: string;
  fee_brl?: number;
  done: boolean;
}

export interface TTContract {
  id: number;
  contract_number: string;
  patent_id?: number | null;
  pool_id?: number | null;
  licensor: string;
  licensee: string;
  licensee_cnpj: string;
  license_kind: LicenseKind;
  sublicensable: boolean;
  territory: string;
  field_of_use: string;
  royalty_rate: number;
  royalty_floor_annual: number;
  upfront_fee: number;
  inventor_share_pct: number;
  milestones: Milestone[];
  signed_at?: string | null;
  expires_at?: string | null;
  status: ContractStatus;
  nit_approved: boolean;
  audit_rights: boolean;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface TTContractListResponse {
  items: TTContract[];
  pagination: { total: number; limit: number; offset: number };
}

// ─── Patent Pools ────────────────────────────────────────────────────────────

export type PoolKind = "offensive" | "defensive" | "standard_essential";
export type PoolStatus = "forming" | "active" | "closed";

export interface PoolMember {
  id: number;
  pool_id: number;
  patent_id: number;
  share_pct: number;
  added_at: string;
  patent_number?: string;
  patent_title?: string;
}

export interface PatentPool {
  id: number;
  name: string;
  description: string;
  pool_kind: PoolKind;
  royalty_rate: number;
  territory: string;
  duration_years: number;
  administrator: string;
  status: PoolStatus;
  created_at: string;
  updated_at: string;
  members?: PoolMember[];
}

export interface PoolListResponse {
  items: PatentPool[];
  count: number;
}

/** Legacy frontend-only shape (still used by mock data). */
export interface LegacyDispute {
  id: string;
  number: string;
  title: string;
  plaintiff: string;
  defendant: string;
  status: "open" | "in_analysis" | "mediation" | "resolved" | "urgent";
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

// ─── Legacy Patent Pool & TT types (mock-only, replaced above) ───────────────

export type LicenseType = "exclusive" | "non-exclusive" | "sub-licensable";

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

export interface LegacyTTContract {
  id: string;
  number: string;
  patent_title: string;
  licensor: string;
  licensee: string;
  status: "active" | "negotiating" | "expired" | "draft";
  signed_at: string;
  expiry_at: string;
  royalty_rate: number;
  milestones: { label: string; due_date: string; done: boolean }[];
  royalties: RoyaltyEntry[];
  blockchain_hash?: string;
}

export interface RoyaltyEntry {
  period: string;
  expected: number;
  received: number | null;
  status: "received" | "pending" | "upcoming";
}

// ─── Academic metrics (AUTM / HJT / Etzkowitz / Lanjouw-Schankerman) ─────────

export interface AUTMHealthScore {
  scope: string;
  patents: number;
  inventors: number;
  p1_disclosures: number;
  p2_grant_rate: number;
  p3_license_rate: number;
  p4_revenue_per_asset: number;
  p5_time_to_grant: number;
  composite_score: number;
  methodology: string;
  benchmarks: Record<string, number>;
}

export interface TTFunnel {
  disclosures: number;
  patents_filed: number;
  patents_granted: number;
  active_contracts: number;
  total_revenue_brl: number;
  rate_disclosure_to_file: number;
  rate_file_to_grant: number;
  rate_grant_to_contract: number;
  methodology: string;
}

export interface HJTDiversity {
  scope: string;
  ipc_categories_present: number;
  diversity_index: number;
  specialization_index: number;
  methodology: string;
}

export interface TripleHelixScore {
  u_count: number;
  i_count: number;
  g_count: number;
  collab_rate: number;
  helix_score: number;
  methodology: string;
}

export interface InventorMetric {
  name: string;
  total_patents: number;
  granted_patents: number;
  h_index_proxy: number;
  ipc_breadth: number;
  department?: string;
}

export interface MetricsResponse {
  health_score: AUTMHealthScore;
  tt_funnel: TTFunnel;
  ipc_diversity: HJTDiversity;
  triple_helix: TripleHelixScore;
  top_inventors: InventorMetric[];
}

export interface MaintenanceRecommendation {
  patent_id: number;
  application_number: string;
  age_years: number;
  remaining_years: number;
  next_annuity_brl: number;
  total_remaining_cost_brl: number;
  revenue_so_far_brl: number;
  active_licenses: number;
  expected_npv_brl: number;
  recommendation: "keep" | "license" | "abandon";
  reasoning: string[];
  confidence: number;
  methodology: string;
}

export interface CoinventorRef {
  name: string;
  co_patent_count: number;
}

export interface InventorPatentRef {
  id: number;
  application_number: string;
  title: string;
  filing_year: number;
  ipc_category: number;
  status: string;
}

export interface InventorProfile {
  name: string;
  total_patents: number;
  granted_patents: number;
  h_index_proxy: number;
  ipc_breadth: number;
  filing_year_span: string;
  estimated_royalty_brl: number;
  coinventors: CoinventorRef[];
  patents: InventorPatentRef[];
  ipc_distribution: Record<string, number>;
  methodology: string;
}

export interface DepartmentHealth {
  department: string;
  patents: number;
  grant_rate: number;
  license_rate: number;
  revenue_per_asset_brl: number;
  composite_score: number;
}

export interface KnowledgePoint {
  year: number;
  new_patents: number;
  knowledge_stock: number;
}

export interface KnowledgeStockResponse {
  series: KnowledgePoint[];
  scope: string;
  methodology: string;
  depreciation_rate: number;
}

// ─── Royalty Forecast (Pakes 1986) ───────────────────────────────────────────

export interface ForecastYear {
  year: number;
  expected_royalty_brl: number;
  cumulative_brl: number;
  active_contracts: number;
  expiring_this_year: number;
  new_contracts_expected: number;
  expected_npv_brl: number;
}

export interface RoyaltyForecast {
  years: ForecastYear[];
  total_projected_brl: number;
  total_npv_brl: number;
  discount_rate: number;
  growth_assumption: string;
  methodology: string;
}

// ─── Smart Filing Assistant ──────────────────────────────────────────────────

export interface FilingPriorArtHit {
  patent_id: number;
  application_number: string;
  title: string;
  applicant: string;
  ipc_category: number;
  similarity_pct: number;
  status: string;
}

export interface FilingSuggestion {
  ipc_category: number;
  ipc_letter: string;
  ipc_name: string;
  ipc_confidence: "high" | "low";
  distinctiveness: number;
  specificity: number;
  novelty_score: number;
  overall_score: number;
  recommendation: "proceed" | "refine" | "not_recommended";
  prior_art_hits: FilingPriorArtHit[];
  suggested_claim: string;
  next_steps: string[];
  methodology: string;
}

export interface PCIScore {
  patent_id: number;
  forward_citations: number;
  backward_citations: number;
  family_size: number;
  claims_count: number;
  pci_score: number;
  methodology: string;
  weights: string;
  has_citation_data: boolean;
  source: "lens" | "mock" | "none";
}

export interface MethodologyComponent {
  key: string;
  label: string;
  definition: string;
}

export interface MethodologyMetric {
  id: string;
  name: string;
  description: string;
  formula: string;
  components?: MethodologyComponent[];
  interpretation?: string;
  normalization?: string;
  data_requirements?: string;
  references: string[];
}

export interface MethodologyPayload {
  version: string;
  metrics: MethodologyMetric[];
}

// ─── TT Marketplace (público) ────────────────────────────────────────────────

export interface MarketplaceListing {
  patent_id: number;
  application_number: string;
  title: string;
  abstract: string;
  inventors: string[];
  filing_year: number;
  ipc_category: number;
  ipc_letter: string;
  ipc_name: string;
  status: string;
  non_exclusive_slots_available: number;
  existing_licensees: number;
  suggested_license_kind: string;
}

export interface MarketplaceResponse {
  items: MarketplaceListing[];
  count: number;
  by_ipc_category: Record<string, number>;
}

// ─── Citation Network (Narin 1994) ───────────────────────────────────────────

export interface CitationNode {
  id: string;
  label: string;
  group: "self" | "forward" | "backward";
  year?: number;
  ipc?: string;
}

export interface CitationLink {
  source: string;
  target: string;
  kind: "forward" | "backward";
}

export interface CitationNetwork {
  nodes: CitationNode[];
  links: CitationLink[];
  center_node_id: string;
  stats: {
    node_count: number;
    forward_count: number;
    backward_count: number;
    avg_year: number;
  };
}

// ─── Global search ───────────────────────────────────────────────────────────

export interface SearchHit {
  kind: "patent" | "trademark" | "dispute" | "contract";
  id: number;
  reference: string;
  title: string;
  subtitle: string;
  url: string;
}

export interface SearchResponse {
  query: string;
  total: number;
  hits: SearchHit[];
}

// ─── Chat threads ────────────────────────────────────────────────────────────

export type ChatRole = "user" | "assistant" | "system";

export interface ChatMessage {
  id: number;
  thread_id: number;
  role: ChatRole;
  content: string;
  created_at: string;
}

export interface ChatThread {
  id: number;
  title: string;
  pinned: boolean;
  archived: boolean;
  message_count: number;
  created_at: string;
  updated_at: string;
  messages?: ChatMessage[];
}

export interface ChatThreadListResponse {
  items: ChatThread[];
  count: number;
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
  auto_dispute: boolean;
  similarity_threshold: number;
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
