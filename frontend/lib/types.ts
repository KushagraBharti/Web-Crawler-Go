export type SourceMode = 'url' | 'keyword';

export type RunConfig = {
  mode: SourceMode;
  input: string;
  seed_url: string;
  max_depth: number;
  max_pages: number;
  time_budget_seconds: number;
  max_links_per_page: number;
  global_concurrency: number;
  per_host_concurrency: number;
  user_agent: string;
  respect_robots: boolean;
};

export type SearchSeed = {
  query: string;
  primary_url: string;
  results: string[];
};

export type RunSummary = {
  status: string;
  pages_fetched: number;
  pages_failed: number;
  pages_queued: number;
  last_fetched_at?: string | null;
};

export type Page = {
  id: string;
  parent_page_id?: string;
  source_mode: string;
  source_input: string;
  url: string;
  canonical_url: string;
  host: string;
  title: string;
  text: string;
  excerpt: string;
  outgoing_links: string[];
  depth: number;
  status_code: number;
  content_type: string;
  fetch_ms: number;
  size_bytes: number;
  error_class?: string;
  error_message?: string;
  discovered_at: string;
  fetched_at: string;
};

export type TreeNode = {
  id: string;
  parent_page_id?: string;
  url: string;
  title: string;
  depth: number;
};

export type TreeEdge = {
  parent_page_id: string;
  child_page_id: string;
};

export type DiagnosticsEntry = {
  at: string;
  url?: string;
  page_id?: string;
  parent_id?: string;
  reason: string;
  detail?: string;
};

export type Diagnostics = {
  search_seed?: SearchSeed;
  skipped_urls: DiagnosticsEntry[];
  retry_events: DiagnosticsEntry[];
  errors: DiagnosticsEntry[];
  fetch_log: DiagnosticsEntry[];
  artifact_dir: string;
  artifact_files: Record<string, string>;
};

export type Snapshot = {
  run_id: string;
  root_page_id?: string;
  config: RunConfig;
  seed: SearchSeed;
  status: string;
  stop_reason?: string;
  created_at: string;
  started_at?: string | null;
  stopped_at?: string | null;
  summary: RunSummary;
  pages: Page[];
  tree_nodes: TreeNode[];
  tree_edges: TreeEdge[];
  diagnostics: Diagnostics;
  paths: Record<string, string>;
};

export type EventFrame = {
  ts: string;
  status: string;
  summary: RunSummary;
  new_pages?: Page[];
  tree_nodes?: TreeNode[];
  tree_edges?: TreeEdge[];
  latest?: Page | null;
};
