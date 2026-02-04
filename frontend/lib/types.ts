export type Frame = {
  ts: string;
  throughput: { pages_per_sec: number };
  queues: { frontier: number; fetch: number; parse: number };
  errors: { class: string; count: number }[];
  hosts: {
    host: string;
    inflight: number;
    p95_ms: number;
    error_rate: number;
    reuse_rate: number;
    robots_state?: string;
    circuit_state?: string;
  }[];
  graph_delta: {
    nodes: string[];
    edges: [string, string, number][];
  };
};

export type RunSummary = {
  pages_fetched: number;
  pages_failed: number;
  unique_hosts: number;
  total_bytes: number;
  last_fetched_at?: string | null;
};

export type PageRow = {
  url: string;
  host: string;
  depth: number;
  status_code: number;
  content_type: string;
  fetch_ms: number;
  size_bytes: number;
  error_class: string;
  error_message: string;
  fetched_at?: string | null;
};
