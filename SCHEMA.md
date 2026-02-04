# Schema

## runs
Stores crawl runs and configuration.

Columns
- id (uuid, pk)
- seed_url (text)
- status (text) values: created, running, stopped, finished, failed
- created_at (timestamptz)
- started_at (timestamptz, nullable)
- stopped_at (timestamptz, nullable)
- max_depth (int)
- max_pages (int)
- time_budget_seconds (int)
- max_links_per_page (int)
- global_concurrency (int)
- per_host_concurrency (int)
- user_agent (text)
- respect_robots (bool)

Indexes
- runs_status_idx (status)

## pages
Metadata per fetched URL.

Columns
- id (bigserial, pk)
- run_id (uuid, fk -> runs.id)
- url (text)
- canonical_url (text)
- host (text)
- depth (int)
- status_code (int, nullable)
- content_type (text, nullable)
- fetch_ms (int, nullable)
- size_bytes (bigint, nullable)
- error_class (text, nullable)
- error_message (text, nullable)
- discovered_at (timestamptz)
- fetched_at (timestamptz, nullable)

Indexes
- pages_run_id_idx (run_id)
- pages_host_idx (run_id, host)
- pages_canonical_idx (run_id, canonical_url)

## hosts
Per-host state for the current run.

Columns
- run_id (uuid, fk -> runs.id)
- host (text)
- robots_state (text) values: unknown, fetching, ready, error
- circuit_state (text) values: closed, open, half_open
- inflight (int)
- last_error_at (timestamptz, nullable)
- last_429_at (timestamptz, nullable)

Primary key
- (run_id, host)

## host_stats
Aggregated per-host stats by time bucket.

Columns
- run_id (uuid, fk -> runs.id)
- host (text)
- bucket_start (timestamptz)
- req_count (int)
- err_count (int)
- p50_ms (int)
- p95_ms (int)
- bytes (bigint)
- reuse_rate (float)

Primary key
- (run_id, host, bucket_start)

## edges
Cross-host link graph edges.

Columns
- run_id (uuid, fk -> runs.id)
- src_host (text)
- dst_host (text)
- count (int)

Primary key
- (run_id, src_host, dst_host)

## errors
Error log for debugging and UI summaries.

Columns
- id (bigserial, pk)
- run_id (uuid, fk -> runs.id)
- host (text, nullable)
- url (text, nullable)
- class (text)
- message (text, nullable)
- at (timestamptz)

Indexes
- errors_run_id_idx (run_id)
- errors_class_idx (run_id, class)