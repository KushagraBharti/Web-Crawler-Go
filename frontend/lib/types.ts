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